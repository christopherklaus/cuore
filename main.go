package main

import (
	"bufio"
	"cuore/common"
	"cuore/config"
	"cuore/integrations/hue"
	"cuore/integrations/sonos"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
)

var Sonos *sonos.Sonos = &sonos.Sonos{ControlPlayers: true}
var Hue *hue.Hue = &hue.Hue{}

func init() {
	config.LoadEnvs()
}

func main() {
	var wg sync.WaitGroup
	wg.Add(2)

	shutdownChan := make(chan struct{})
	defer close(shutdownChan)

	go apiRouter(&wg, shutdownChan)
	go mqttBroker(&wg, shutdownChan)

	wg.Wait()
}

func apiRouter(wg *sync.WaitGroup, shutdownChan <-chan struct{}) {
	defer wg.Done()
	r := gin.Default()

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"code": 404, "message": "Page not found"})
	})

	var sonosRoutes *gin.RouterGroup = r.Group("/integrations/sonos")
	var hueRoutes *gin.RouterGroup = r.Group("/integrations/hue")
	Sonos.AuthorizationHandlers(sonosRoutes)
	Hue.AuthorizationHandlers(hueRoutes)

	err := http.ListenAndServe(fmt.Sprintf(":%d", 80), r)
	if err != nil {
		log.Print(err)
	}

	<-shutdownChan
	log.Print("Shutting down API router")
}

func messagePubHandler(client mqtt.Client, msg mqtt.Message) {
	var messageData common.Page

	log.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())

	if err := json.Unmarshal(msg.Payload(), &messageData); err != nil {
		log.Fatalf("Error decoding JSON: %v", err)
		return
	}

	switch messageData.Name {
	case "Music":
		Sonos.UpdateState(messageData)
		return
	case "Lights":
		Hue.UpdateState(messageData)
		return
	default:
		log.Printf("Unknown device: %s", messageData.Name)
	}
}

func mqttBroker(wg *sync.WaitGroup, shutdownChan <-chan struct{}) {
	defer wg.Done()
	opts := mqtt.NewClientOptions().AddBroker(config.Get().MQTTServer)
	// TODO: move to Config
	opts.SetClientID("cuore")

	c := mqtt.NewClient(opts)

	//we are going to try connecting for max 10 times to the server if the connection fails.
	for i := 0; i < 10; i++ {
		if token := c.Connect(); token.Wait() && token.Error() == nil {
			break
		} else {
			log.Print(token.Error())
			time.Sleep(1 * time.Second)
		}
	}

	if token := c.Subscribe("status/button", 0, messagePubHandler); token.Wait() && token.Error() != nil {
		log.Print(token.Error())
		os.Exit(1)
	}

	for {
		var message string
		// reading new message from console
		fmt.Print("Message to sent, format {}: ")
		reader := bufio.NewReader(os.Stdin)
		message, err := reader.ReadString('\n')
		if err != nil {
			log.Print(err)
		}
		if strings.Compare(message, "\n") > 0 {
			// if there is a message, publish it.
			token := c.Publish("status/server", 0, false, message)
			if strings.Compare(message, "bye\n") == 0 {
				// if message "bye" then exit the shell.
				break
			}
			token.Wait()
		}
	}

	<-shutdownChan

	log.Print("Shutting down MQTT broker")
	if token := c.Unsubscribe("status/button"); token.Wait() && token.Error() != nil {
		log.Print(token.Error())
		os.Exit(1)
	}

	c.Disconnect(250)
}
