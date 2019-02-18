package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	keyring "github.com/zalando/go-keyring"
	"golang.org/x/net/html"
	"golang.org/x/net/publicsuffix"
)

var target string
var username string
var keyringUser string
var keyringService string
var allTokens bool

const (
	previewLoginURL    string = "https://auth.prod-preview.openshift.io/api/logout?redirect=https%3A%2F%2Fapi.prod-preview.openshift.io%2Fapi%2Flogin%2Fauthorize%3Fredirect%3Dhttps%253A%252F%252Fprod-preview.openshift.io%252F"
	productionLoginURL string = "https://auth.openshift.io/api/logout?redirect=https%3A%2F%2Fapi.openshift.io%2Fapi%2Flogin%2Fauthorize%3Fredirect%3Dhttps%253A%252F%252Fopenshift.io%252F"
)

// NewLoginCommand a command to login on `fabric8-auth` service
func newLoginCommand() *cobra.Command {
	loginCmd := &cobra.Command{
		Short: "obtain an access token and a refresh token for preview or production platforms",
		Use:   "login",
		Run:   login,
	}
	loginCmd.Flags().StringVarP(&target, "target", "t", "preview", "the target platform to log in: 'preview' or 'production'")
	loginCmd.Flags().StringVarP(&username, "username", "u", "", "your username (optional. Will use the keyring-user if missing)")
	loginCmd.Flags().BoolVarP(&allTokens, "all-tokens", "a", true, "return refresh token and access token")
	loginCmd.Flags().StringVarP(&keyringUser, "keyring-user", "", "", "Keyring user")
	loginCmd.Flags().StringVarP(&keyringService, "keyring-service", "", "", "Keyring service")
	return loginCmd
}

func login(cmd *cobra.Command, args []string) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Fatal(err)
	}

	// If keyringUser is specified but not username then let's have the username as the keyringUser
	if keyringUser != "" && username == "" {
		username = keyringUser
	}

	// if the username is empty but the keyring account has been provided, then use it as-is
	if username == "" && keyringUser != "" {
		username = keyringUser
	} else if username == "" {
		// prompt for username and password
		fmt.Print("username: ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			username = strings.TrimRight(scanner.Text(), "\r\n")
		}
	}

	// try to get password from keyring, it's okay if we can't find it (i.e: ignore error)
	password, _ := keyring.Get(keyringService, keyringUser)

	if password == "" {
		fmt.Print("password: ")
		b, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			log.Fatal(err)
		}
		password = string(b)
	}
	var targetURL string
	switch target {
	case "production":
		targetURL = productionLoginURL
	default:
		targetURL = previewLoginURL

	}
	// submit login form
	if verbose {
		fmt.Println("\nperforming login, please wait... 😴")
	}
	action, method, err := retrieveLoginActionURL(targetURL, jar)
	if err != nil {
		log.Fatal(err)
	}
	landingURL, err := performLogin(action, method, username, password, jar)
	if err != nil {
		log.Fatal(err)
	}
	// now, extract the `token_json` query param from the landing page URL
	if tokenJSON, ok := landingURL.Query()["token_json"]; ok {
		if len(tokenJSON) == 0 {
			log.Fatalf("failed to parse the tokens retrieved upon login: the 'token_json' quuery param of the landing URL is empty: %s", landingURL.String())
		}
		// unmarshal
		tokens := Tokens{}
		err := json.Unmarshal([]byte(tokenJSON[0]), &tokens)
		if err != nil {
			log.Fatalf("failed to parse the tokens retrieved upon login: %v", err)
		}
		if allTokens {
			fmt.Printf(`{
	"access-token": "%s",
	"refresh-token": "%s"
}`, tokens.AccessToken, tokens.RefreshToken)
		} else {
			fmt.Printf(tokens.AccessToken)
		}
	} else {
		log.Fatalf("the landing URL did not contain a 'token_json' query param: %s", landingURL.String())

	}
}

// Tokens the access and refresh tokens received upon login
type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type traceTransport struct {
	t               http.RoundTripper
	currentLocation *url.URL
}

func newTraceTransport() *traceTransport {
	return &traceTransport{
		t: http.DefaultTransport,
	}
}

func (t *traceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.t.RoundTrip(req)
	log.Debugf("request URI: %s", req.URL)
	log.Debugf("request body: %s", req.Body)
	log.Debugf("request headers:")
	for k, v := range req.Header {
		log.Debugf("%s: %v", k, v)
	}
	// log.Infof("request cookie: %s", req.Cookies())
	log.Debugf("request referer: %s", req.Referer())
	log.Debugf("response status: %s", resp.Status)
	log.Debugf("response headers:")
	for k, v := range resp.Header {
		log.Debugf("%s: %v", k, v)
	}
	t.currentLocation = req.URL
	return resp, err
}

// performLogin performs the login and returns the landing URL
func performLogin(loginURL, method, username, password string, jar *cookiejar.Jar) (*url.URL, error) {
	t := newTraceTransport()
	client := &http.Client{
		Jar:       jar,
		Transport: t,
	}
	form := url.Values{}
	form.Add("username", username)
	form.Add("password", password)
	form.Add("login", "Log in")
	req, err := http.NewRequest(strings.ToUpper(method), loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, errors.Wrap(err, "failed to perform HTTP request")
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to perform HTTP request")
	}
	return t.currentLocation, nil

}

// follows the given loginURL to reach an HTML page that contains a login form with `id="kc-form-login"` and retrieves its associated `action` attribute
func retrieveLoginActionURL(loginURL string, jar *cookiejar.Jar) (string, string, error) {
	req, err := http.NewRequest("GET", loginURL, nil)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to perform HTTP request")
	}
	client := &http.Client{
		Jar: jar,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to perform HTTP request")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to retrieve the HTML login page")
	}
	// fmt.Println(string(body))
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return "", "", errors.Wrap(err, "failed to parse the HTML login page")
	}
	var action string
	var method string
	var f func(*html.Node)
	f = func(n *html.Node) {
		found := false
		if n.Type == html.ElementNode && n.Data == "form" {
			for _, a := range n.Attr {
				if a.Key == "id" && a.Val == "kc-form-login" {
					found = true
				}
				if a.Key == "action" {
					action = a.Val
				}
				if a.Key == "method" {
					method = a.Val
				}
			}
		}
		if !found {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}
	}
	f(doc)
	if action == "" {
		return "", "", errors.New("failed to locate the login form in the HTML page")
	}
	return action, method, nil
}
