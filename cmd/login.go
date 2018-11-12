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
	"golang.org/x/net/html"
	"golang.org/x/net/publicsuffix"
)

// NewLoginCommand a command to login on `fabtic8-auth` service
func newLoginCommand() *cobra.Command {
	c := &cobra.Command{
		Short: "login",
		Use:   "login",
		// Args:  cobra.MinimumNArgs(1),
		Run: login,
	}

	return c
}

func login(cmd *cobra.Command, args []string) {
	// login
	// redirects to https://auth.prod-preview.openshift.io/api/logout?redirect=https%3A%2F%2Fapi.prod-preview.openshift.io%2Fapi%2Flogin%2Fauthorize%3Fredirect%3Dhttps%253A%252F%252Fprod-preview.openshift.io%252F
	// redirects to https://auth.prod-preview.openshift.io/api/login?redirect=https%3A%2F%2Fprod-preview.openshift.io%2F
	// redirects to https://sso.prod-preview.openshift.io/auth/realms/fabric8/protocol/openid-connect/auth?access_type=online&client_id=fabric8-online-platform&redirect_uri=https%3A%2F%2Fauth.prod-preview.openshift.io%2Fapi%2Flogin&response_type=code&scope=user%3Aemail&state=... with custom state and cookie
	// see other: https://sso.prod-preview.openshift.io/auth/realms/fabric8/broker/rhd/login?code=y...&client_id=fabric8-online-platform with `AUTH_SESSION_ID` cookie
	// see other https://developers.redhat.com/auth/realms/rhd/protocol/openid-connect/auth?scope=openid&state=...&response_type=code&client_id=fabric8-online&redirect_uri=https%3A%2F%2Fsso.prod-preview.openshift.io%2Fauth%2Frealms%2Ffabric8%2Fbroker%2Frhd%2Fendpoint

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Fatal(err)
	}
	action, method, err := retrieveLoginActionURL("https://auth.prod-preview.openshift.io/api/logout?redirect=https%3A%2F%2Fapi.prod-preview.openshift.io%2Fapi%2Flogin%2Fauthorize%3Fredirect%3Dhttps%253A%252F%252Fprod-preview.openshift.io%252F", jar)
	if err != nil {
		log.Fatal(err)
	}
	// prompt for username and password
	fmt.Print("username: ")
	scanner := bufio.NewScanner(os.Stdin)
	var username string
	if scanner.Scan() {
		username = strings.TrimRight(scanner.Text(), "\r\n")
	}
	fmt.Print("password: ")
	password, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	// submit login form
	fmt.Println("\nperforming login, please wait... 😴")
	landingURL, err := performLogin(action, method, username, password, jar)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("landed on %s", landingURL)
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
		fmt.Printf("access token: %s\n\n", tokens.AccessToken)
		fmt.Printf("refresh token: %s\n\n", tokens.RefreshToken)
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
func performLogin(loginURL, method, username string, password []byte, jar *cookiejar.Jar) (*url.URL, error) {
	t := newTraceTransport()
	client := &http.Client{
		Jar:       jar,
		Transport: t,
	}
	form := url.Values{}
	form.Add("username", username)
	form.Add("password", string(password))
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
