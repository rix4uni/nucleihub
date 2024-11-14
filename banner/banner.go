package banner

import (
	"fmt"
)

// prints the version message
const version = "v0.0.3"

func PrintVersion() {
	fmt.Printf("Current nucleihub version %s\n", version)
}

// Prints the Colorful banner
func PrintBanner() {
	banner := `
                        __       _  __            __  
   ____   __  __ _____ / /___   (_)/ /_   __  __ / /_ 
  / __ \ / / / // ___// // _ \ / // __ \ / / / // __ \
 / / / // /_/ // /__ / //  __// // / / // /_/ // /_/ /
/_/ /_/ \__,_/ \___//_/ \___//_//_/ /_/ \__,_//_.___/
`
	fmt.Printf("%s\n%60s\n\n", banner, "Current nucleihub version "+version)
}
