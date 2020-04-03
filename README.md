# SMTPD 

Simple smtp server library for GO. Each received message will call the handler, which for example can upload the message to a webservice. (See example)

## Implementation

```
package main

import (
		"context"
        "fmt"
		"log"
        "os"

        "github.com/dutchcoders/smtpd"
)

func main() {
        smtpd.HandleFunc(func(msg smtpd.Message) error {
                fmt.Printf("%#v\n", msg)
                return nil
        })

		listener := smtpd.NewListener(
			smtpd.ListenWithPort(os.Getenv("PORT")),
		)

		server, err := smtpd.New(
			smtpd.WithListener(listener),
		)
		if err != nil {
			panic(err)
		}

        log.Fatal(server.ListenAndServe(context.Background()))
}
```

## Contributions

Contributions are welcome.

## Creators

**Remco Verhoef**
- <https://twitter.com/remco_verhoef>
- <https://twitter.com/dutchcoders>

## Copyright and license

Code and documentation copyright 2011-2015 Remco Verhoef.

Code released under [the MIT license](LICENSE).
