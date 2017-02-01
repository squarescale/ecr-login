package main

import (
	"encoding/base64"
	"flag"
	"log"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/coreos/go-systemd/daemon"
)

// Auth structure
type Auth struct {
	Token         string
	User          string
	Pass          string
	ProxyEndpoint string
	ExpiresAt     time.Time
}

// error handler
func check(e error) {
	if e != nil {
		log.Panic(e)
	}
}

// DefaultTemplate prints docker login command
const DefaultTemplate = `{{range .}}docker login -u {{.User}} -p {{.Pass}} -e none {{.ProxyEndpoint}}
{{end}}`

// load template from file or use default
func getTemplate() *template.Template {
	var tmpl *template.Template
	var err error

	file, exists := os.LookupEnv("TEMPLATE")

	if exists {
		tmpl, err = template.ParseFiles(file)
	} else {
		tmpl, err = template.New("default").Parse(DefaultTemplate)
	}

	check(err)
	return tmpl
}

// if AWS_REGION not set, infer from instance metadata
func getRegion(sess *session.Session) string {
	region, exists := os.LookupEnv("AWS_REGION")
	if !exists {
		ec2region, err := ec2metadata.New(sess).Region()
		check(err)
		region = ec2region
	}
	return region
}

// get list of registries from env, leave empty for default
func getRegistryIds() []*string {
	var registryIds []*string
	registries, exists := os.LookupEnv("REGISTRIES")
	if exists {
		for _, registry := range strings.Split(registries, ",") {
			registryIds = append(registryIds, aws.String(registry))
		}
	}
	return registryIds
}

func login() ([]Auth, error) {
	// configure aws client
	sess := session.New()
	svc := ecr.New(sess, aws.NewConfig().WithRegion(getRegion(sess)))

	// this lets us handle multiple registries
	params := &ecr.GetAuthorizationTokenInput{
		RegistryIds: getRegistryIds(),
	}

	// request the token
	resp, err := svc.GetAuthorizationToken(params)
	if err != nil {
		return nil, err
	}

	// fields to send to template
	fields := make([]Auth, len(resp.AuthorizationData))
	for i, auth := range resp.AuthorizationData {

		// extract base64 token
		data, err := base64.StdEncoding.DecodeString(*auth.AuthorizationToken)
		if err != nil {
			return nil, err
		}

		// extract username and password
		token := strings.SplitN(string(data), ":", 2)

		// object to pass to template
		fields[i] = Auth{
			Token:         *auth.AuthorizationToken,
			User:          token[0],
			Pass:          token[1],
			ProxyEndpoint: *(auth.ProxyEndpoint),
			ExpiresAt:     *(auth.ExpiresAt),
		}
	}

	return fields, nil
}

func main() {
	var renew bool
	flag.BoolVar(&renew, "renew", false, "Stay in foreground to renew credentials. You need to share /var/run/docker.sock and /usr/bin/docker")
	flag.Parse()

	if renew {
		for {
			log.Println("Renew AWS crendentials...")
			fields, err := login()
			check(err)
			expires := fields[0].ExpiresAt
			for i, cred := range fields {
				log.Printf("[%d/%d] Got credentials expiring at %v", i+1, len(fields), cred.ExpiresAt.String())
				cmd := exec.Command("/usr/bin/docker", "login", "-u", cred.User, "-p", cred.Pass, "-e", "none", cred.ProxyEndpoint)
				err = cmd.Run()
				check(err)
				if cred.ExpiresAt.Before(expires) {
					expires = cred.ExpiresAt
				}
			}
			_, err = daemon.SdNotify(true, "READY=1")
			if err != nil {
				log.Printf("sd_notify(READY=1): %v", err)
			}
			expires = expires.Add(-time.Hour)
			log.Printf("Schedule next login for %v", expires.String())
			time.Sleep(expires.Sub(time.Now()))
		}
	} else {
		fields, err := login()
		check(err)

		// run the template
		err = getTemplate().Execute(os.Stdout, fields)
		check(err)
	}
}
