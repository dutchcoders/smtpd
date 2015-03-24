# SMTPD 

Simple smtp server library for GO. Each received message will call the handler, which for example can upload the message to a webservice. (See example)

## Implementation

```
package main

import (
        "github.com/dutchcoders/smtpd"
        "os"
        "fmt"
)

func main() {
        smtpd.HandleFunc(func(msg smtpd.Message) error {
                fmt.Printf("%#v\n", msg)
                return nil
        })

        addr := fmt.Sprintf(":%s", os.Getenv("PORT"))
        smtpd.ListenAndServe(addr)
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
