package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/mail"
	"time"

	"github.com/dutchcoders/smtpd"
)

func main() {
	receiveChan := make(chan smtpd.Message)

	go func() {
		for {
			select {
			case message := <-receiveChan:
				log.Println("Message received", message)
				type Address struct {
					Name    string `json:"name"`
					Address string `json:"address"`
				}

				var j struct {
					From        Address     `json:"from"`
					To          []Address   `json:"to"`
					AddressList []Address   `json:"address_list"`
					Date        *time.Time  `json:"date"`
					Headers     mail.Header `json:"header"`
					Body        []byte      `json:"body"`
				}

				var err error
				j.From = Address{Name: message.From.Name, Address: message.From.Address}

				for _, to := range message.To {
					j.To = append(j.To, Address{Name: to.Name, Address: to.Address})
				}

				j.Headers = message.Header

				j.Body, err = ioutil.ReadAll(message.Body)
				if err != nil {
					log.Println(err)
					continue
				}

				var b []byte
				b, err = json.Marshal(j)
				if err != nil {
					log.Println(err)
					continue
				}

				_ = b

				fmt.Println(string(b))
			}
		}
	}()

	//set default Handler.
	_ = smtpd.HandleFunc(func(msg smtpd.Message) error {
		receiveChan <- msg
		return nil
	})

	//uses the default Handler.
	l1 := smtpd.Listener{
		ID:      "Listener 1",
		Address: "127.0.0.1",
		Port:    "8025",
		Mode:    "plain",
		Banner:  func() string { return "SMTPd Listener 1" },
	}

	mux := smtpd.NewServeMux()
	mux.HandleFunc(func(msg smtpd.Message) error {
		fmt.Println("HandleFunc: Listener 2")
		receiveChan <- msg
		return nil
	})

	//uses a custom Handler.
	l2 := smtpd.Listener{
		ID:      "Listener 2",
		Address: "127.0.0.1",
		Port:    "8026",
		Mode:    "plain",
		Banner:  func() string { return "SMTPd TLS Listener 2" },
		Handler: mux,
	}

	server, err := smtpd.New(
		smtpd.WithListener(l1),
		smtpd.WithListener(l2),
	)
	if err != nil {
		panic(err)
	}

	//start listening on configured ports.
	log.Println(server.ListenAndServe(context.Background()))
}
