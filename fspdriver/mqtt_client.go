/*
 * Copyright (c) 2021 IBM Corp and others.
 *
 * All rights reserved. This program and the accompanying materials
 * are made available under the terms of the Eclipse Public License v2.0
 * and Eclipse Distribution License v1.0 which accompany this distribution.
 *
 * The Eclipse Public License is available at
 *    https://www.eclipse.org/legal/epl-2.0/
 * and the Eclipse Distribution License is available at
 *   http://www.eclipse.org/org/documents/edl-v10.php.
 *
 * Contributors:
 *    Seth Hoenig
 *    Allan Stockdill-Mander
 *    Mike Robertson
 */

package fspdriver

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"gocv.io/x/gocv"
)

var f mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

func NewMQTTClient() mqtt.Client {
	// mqtt.DEBUG = log.New(os.Stdout, "", 0)
	// mqtt.ERROR = log.New(os.Stdout, "", 0)
	opts := mqtt.NewClientOptions().AddBroker("tcp://192.168.1.57:1883").SetClientID("gotrivial")
	opts.SetKeepAlive(2 * time.Second)
	opts.SetDefaultPublishHandler(f)
	opts.SetPingTimeout(1 * time.Second)

	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	if token := c.Subscribe("go-mqtt/sample", 0, nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	return c
}

func publishImage(topic string, mat gocv.Mat, mqttClient mqtt.Client) error {
	// Publish image (jpg/base64)
	imgBuf, err := gocv.IMEncode(gocv.JPEGFileExt, mat)
	if err != nil {
		return err
	}
	imgBytes := imgBuf.GetBytes()
	var b64bytes []byte = make([]byte, base64.StdEncoding.EncodedLen(len(imgBytes)))
	base64.StdEncoding.Encode(b64bytes, imgBytes)
	mqttClient.Publish(topic, 2, false, b64bytes)
	return err
}

func publishJsonMsg(topic string, obj interface{}, mqttClient mqtt.Client) error {
	msg, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	mqttClient.Publish(topic, 2, false, msg)
	return err
}
