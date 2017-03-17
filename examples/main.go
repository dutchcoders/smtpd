package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/mail"
	"os"
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

	handler := smtpd.HandleFunc(func(msg smtpd.Message) error {
		receiveChan <- msg
		return nil
	})

	server := smtpd.New(
		smtpd.ListenAddr(fmt.Sprintf(":%s", os.Getenv("PORT"))),
	)

	server.ListenAndServe(handler)
}
