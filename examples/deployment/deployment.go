package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"io/ioutil"
	"crypto/tls"
	"encoding/json"
	"strings"
)

var (
	version  string
	subtitle string
	color    string
	cloud    string
	zone     string
)

func deploymentHandler(w http.ResponseWriter, r *http.Request) {
	htmlContent := `<!DOCTYPE html>
	<html lang="en">
	<head>
	  <meta charset="utf-8">
	  <title>Deployment Demonstration</title>
	  <meta name="viewport" content="width=device-width, initial-scale=1.0">
	  <style>
		HTML{height:100%%;}
		BODY{font-family:Helvetica,Arial;display:flex;display:-webkit-flex;align-items:center;justify-content:center;-webkit-align-items:center;-webkit-box-align:center;-webkit-justify-content:center;height:100%%;}
		.box{background:%[3]s;color:white;text-align:center;border-radius:10px;display:inline-block;}
		H1{font-size:10em;line-height:1.5em;margin:0 0.5em;}
		H2{margin-top:0;}
	  </style>
	</head>
	<body>
		<div class="box">
			<h3>Cloud Provider: %[4]s</h3>
			<h3>Zone: %[5]s</h3>
			<h1>%[1]s</h1><h2>%[2]s</h2>
		</div>
	</body>
	</html>`
	fmt.Fprintf(w, htmlContent, version, subtitle, color, cloud, zone)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "ok")
}

func main() {
	version = "v1"
	if len(os.Args) > 1 {
		version = os.Args[1]
	}
	subtitle = os.Getenv("SUBTITLE")
	color = os.Getenv("COLOR")
	if len(color) == 0 {
		color = "#303030"
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify : true},
	}

	client := &http.Client{
		Transport: tr,
	}

	dat, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
    token := fmt.Sprintf(strings.TrimSuffix(string(dat), "\n"))
	server := os.Getenv("KUBERNETES_SERVICE_HOST")
	port := os.Getenv("KUBERNETES_PORT_443_TCP_PORT")
	node := os.Getenv("NODE_NAME")
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s:%s/api/v1/nodes/%s", server, port, node), nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := client.Do(req)
	if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		var result map[string]interface{}
		json.Unmarshal(bodyBytes, &result)
		labels := result["metadata"].(map[string]interface{})["labels"].(map[string]interface{})
		zone := labels["topology.kubernetes.io/zone"].(string)
		spec := result["spec"].(map[string]interface{})
		providerID := spec["providerID"].(string)
		cloud := providerID[0:strings.Index(providerID, ":")]
	} else {
		log.Printf("Unable to retrieve node info! Received %d", resp.StatusCode)
	}

	http.HandleFunc("/", deploymentHandler)

	http.HandleFunc("/_healthz", healthHandler)

	log.Printf("Listening on :8080 at %s ...", version)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
