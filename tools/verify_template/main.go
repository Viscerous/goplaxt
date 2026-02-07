package main

import (
	"fmt"
	"html/template"
	"os"

	"github.com/viscerous/goplaxt/lib/store"
)

type AuthorisePage struct {
	SelfRoot    string
	Authorised  bool
	URL         string
	User        store.User
	CurrentStep int // 1, 2, 3, 4 (Dashboard)
}

func main() {
	// Ensure test_output directory exists
	os.MkdirAll("../../test_output", 0755)
	os.MkdirAll("test_output", 0755)

	outputFile, err := os.Create("../../test_output/template_cases.txt")
	if err != nil {
		outputFile, err = os.Create("test_output/template_cases.txt")
		if err != nil {
			panic(err)
		}
	}
	defer outputFile.Close()

	// Helper for pointers
	boolPtr := func(b bool) *bool { return &b }

	// Prepare template
	tpl, err := template.ParseFiles("../../static/index.html")
	if err != nil {
		tpl, err = template.ParseFiles("../static/index.html")
		if err != nil {
			tpl, err = template.ParseFiles("static/index.html")
			if err != nil {
				panic(err)
			}
		}
	}

	// Case 1: Unauthorised (Step 1)
	fmt.Fprintln(outputFile, "--- Case 1: Unauthorised (Step 1) ---")
	err = tpl.Execute(outputFile, AuthorisePage{
		Authorised:  false,
		CurrentStep: 1,
	})
	if err != nil {
		panic(err)
	}

	// Case 2: Authorised, No Config (Step 2)
	fmt.Fprintln(outputFile, "\n\n--- Case 2: Authorised, No Config (Step 2) ---")
	err = tpl.Execute(outputFile, AuthorisePage{
		Authorised:  true,
		CurrentStep: 2,
		User: store.User{
			Username: "testuser",
			// Config fields are nil -> UI suggestions show as checked via getters
		},
	})
	if err != nil {
		panic(err)
	}

	// Case 3: Authorised, Configured (Dashboard)
	fmt.Fprintln(outputFile, "\n\n--- Case 3: Authorised, Configured (Step 4 - Dashboard) ---")
	err = tpl.Execute(outputFile, AuthorisePage{
		Authorised:  true,
		CurrentStep: 4,
		User: store.User{
			Username:     "testuser",
			PlexUsername: "plexuser",
			Config: store.Config{
				MovieScrobbleStart:   boolPtr(true),
				MovieScrobbleStop:    boolPtr(true),
				MovieRate:            boolPtr(true),
				EpisodeScrobbleStart: boolPtr(true),
				EpisodeScrobbleStop:  boolPtr(true),
				EpisodeRate:          boolPtr(true),
			},
		},
	})
	if err != nil {
		panic(err)
	}

	// Case 4: Authorised, Partial Config (Returning User)
	fmt.Fprintln(outputFile, "\n\n--- Case 4: Authorised, Partial Config (Step 2) ---")
	err = tpl.Execute(outputFile, AuthorisePage{
		Authorised:  true,
		CurrentStep: 2,
		User: store.User{
			Username: "testuser",
			Config: store.Config{
				MovieScrobbleStart: boolPtr(false), // Explicit false -> unchecked
			},
		},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("Verification results written to test_output/template_cases.txt")
}
