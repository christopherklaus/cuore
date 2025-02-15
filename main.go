package main

import (
	"cuore/common"
	"cuore/config"
	"cuore/integrations/hue"
	"cuore/integrations/sonos"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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

func handleControlMessage(msg common.ControlMessage) {
	var err error
	switch msg.Target {
	case "music":
		err = Sonos.HandleControl(msg)
	case "light":
		err = Hue.HandleControl(msg)
	default:
		log.Printf("Unknown target type: %s", msg.Target)
		return
	}

	if err != nil {
		log.Printf("Error handling control message: %v", err)
	}
}

func handleSetupMessage(msg common.SetupMessage) {
	var err error
	switch msg.Target {
	case "music":
		err = Sonos.HandleSetup(msg)
	case "light":
		err = Hue.HandleSetup(msg)
	default:
		log.Printf("Unknown target type: %s", msg.Target)
		return
	}

	if err != nil {
		log.Printf("Error handling setup message: %v", err)
	}
}

func controlMessageHandler(client mqtt.Client, msg mqtt.Message) {
	var controlMsg common.ControlMessage
	if err := json.Unmarshal(msg.Payload(), &controlMsg); err != nil {
		log.Printf("Error decoding control message: %v", err)
		return
	}
	handleControlMessage(controlMsg)
}

func setupMessageHandler(client mqtt.Client, msg mqtt.Message) {
	var setupMsg common.SetupMessage
	if err := json.Unmarshal(msg.Payload(), &setupMsg); err != nil {
		log.Printf("Error decoding setup message: %v", err)
		return
	}
	handleSetupMessage(setupMsg)
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

	if token := c.Subscribe("control", 0, controlMessageHandler); token.Wait() && token.Error() != nil {
		log.Print(token.Error())
		os.Exit(1)
	}
	if token := c.Subscribe("setup", 0, setupMessageHandler); token.Wait() && token.Error() != nil {
		log.Print(token.Error())
		os.Exit(1)
	}

	<-shutdownChan

	log.Print("Shutting down MQTT broker")
	if token := c.Unsubscribe("control"); token.Wait() && token.Error() != nil {
		log.Print(token.Error())
		os.Exit(1)
	}
	if token := c.Unsubscribe("setup"); token.Wait() && token.Error() != nil {
		log.Print(token.Error())
		os.Exit(1)
	}

	c.Disconnect(250)
}
