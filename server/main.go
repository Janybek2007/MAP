package main

import "log"

func main() {
	if err := Run(appFS); err != nil {
		log.Fatal(err)
	}
}
