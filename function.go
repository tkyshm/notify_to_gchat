package function

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"os"
)

var webhookURL = os.Getenv("WEBHOOK_URL")

type AlertLevel int

const (
	Danger AlertLevel = iota
	Warn
	Health
)

var colors = map[AlertLevel]string{
	Danger: "#fc2f2f",
	Warn:   "#ffcc14",
	Health: "#27d871",
}

// {
//   "incident": {
//     "incident_id": "f2e08c333dc64cb09f75eaab355393bz",
//     "resource_id": "i-4a266a2d",
//     "resource_name": "webserver-85",
//     "state": "open",
//     "started_at": 1385085727,
//     "ended_at": null,
//     "policy_name": "Webserver Health",
//     "condition_name": "CPU usage",
//     "url": "https://app.google.stackdriver.com/incidents/f333dc64z",
//     "summary": "CPU for webserver-85 is above the threshold of 1% with a value of 28.5%"
//   },
//   "version": 1.1
// }

type Incident struct {
	IncidentID    string `json:"incident_id"`
	ResourceID    string `json:"resource_id"`
	ResourceName  string `json:"resource_name"`
	State         string `json:"state"`
	StartedAt     int64  `json:"started_at"`
	EndedAt       int64  `json:"ended_at"`
	PolicyName    string `json:"policy_name"`
	ConditionName string `json:"condition_name"`
	URL           string `json:"url"`
	Summary       string `json:"summary"`
}

type Alert struct {
	Incident Incident `json:"incident"`
	Version  string   `json:"version"`
}

//{
//  "text": "",
//  "cards": [
//    {
//      "sections": [
//        {
//          "widgets": [
//            {
//              "textParagraph": {
//                "text": "<b>Roses</b> are <font color=\"#ff0000\">red</font>,<br><i>Violets</i> are <font color=\"#0000ff\">blue</font>"
//              }
//            }
//          ]
//        }
//      ]
//    }
//  ]
//}

type GChatParam struct {
	Text  string `json:"text"`
	Cards []Card `json:"cards"`
}

type Card struct {
	Sections []Section `json:"sections"`
}

type Section struct {
	Widgets []Widget `json:"widgets"`
}

type Widget map[string]interface{}

func NotifyToGChat(w http.ResponseWriter, r *http.Request) {
	alert := Alert{}

	if err := json.NewDecoder(r.Body).Decode(&alert); err != nil {
		log.Printf("[error] decode alert error, %s.", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("[alert log] %v", alert)

	var mention = "<users/all>"
	var color = colors[Warn]
	if strings.HasPrefix(alert.Incident.ConditionName, "[DANGER]") {
		mention = "<users/all>"
		color = colors[Danger]
	}
	if alert.Incident.State == "closed" {
		color = colors[Health]
	}

	text := fmt.Sprintf("<font color=\"%s\">%s %s</font>",
		color,
		alert.Incident.PolicyName,
		alert.Incident.ConditionName,
	)

	headWidgets := []Widget{
		Widget{
			"textParagraph": map[string]interface{}{
				"text": text,
			},
		},
		Widget{
			"buttons": []map[string]interface{}{
				map[string]interface{}{
					"textButton": map[string]interface{}{
						"text": "URL",
						"onClick": map[string]interface{}{
							"openLink": map[string]interface{}{
								"url": alert.Incident.URL,
							},
						},
					},
				},
			},
		},
	}

	var resourceID string
	if alert.Incident.ResourceID == "" {
		resourceID = "-"
	} else {
		resourceID = alert.Incident.ResourceID
	}

	keyValueSecWidgets := []Widget{
		Widget{
			"keyValue": map[string]interface{}{
				"topLabel":         "State",
				"content":          alert.Incident.State,
				"contentMultiline": true,
			},
		},
		Widget{
			"keyValue": map[string]interface{}{
				"topLabel":         "Resoucrce ID",
				"content":          resourceID,
				"contentMultiline": true,
			},
		},
		Widget{
			"keyValue": map[string]interface{}{
				"topLabel":         "Resoucrce Name",
				"content":          alert.Incident.ResourceName,
				"contentMultiline": true,
			},
		},
	}

	params := GChatParam{
		Text: mention,
		Cards: []Card{
			{
				Sections: []Section{
					{
						Widgets: headWidgets,
					},
					{
						Widgets: keyValueSecWidgets,
					},
				},
			},
		},
	}

	b, err := json.Marshal(params)
	if err != nil {
		log.Printf("[error] marshal error, %s.", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(b))
	if err != nil {
		log.Printf("[error] new request error, %s.", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[error] post form error, %s.", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("%s", err.Error())

	} else {
		log.Printf("[gchat response] %s", body)

	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(b); err != nil {
		log.Printf("[error] write error, %s.", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
