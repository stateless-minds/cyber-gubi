package main

import (
	"log"
	"net/http"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type Descriptor struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type RequestBody struct {
	Descriptor []Descriptor `json:"descriptor"`
}

// The main function is the entry point where the app is configured and started.
// It is executed in 2 different environments: A client (the web browser) and a
// server.
func main() {
	// The first thing to do is to associate the wallet component with a path.
	//
	// This is done by calling the Route() function,  which tells go-app what
	// component to display for a given path, on both client and server-side.
	app.Route("/", func() app.Composer { return &home{} })
	app.Route("/auth", func() app.Composer { return &auth{} })
	app.Route("/wallet", func() app.Composer { return &wallet{} })
	app.Route("/wallets", func() app.Composer { return &wallets{} })
	app.Route("/transactions", func() app.Composer { return &transactions{} })
	app.RouteWithRegexp("/transactions/(?P<uuid>[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})", func() app.Composer { return &transactions{} })
	app.RouteWithRegexp("/tax/(?P<uuid>[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})", func() app.Composer { return &tax{} })
	app.Route("/payment", func() app.Composer { return &payment{} })
	app.Route("/subscriptions", func() app.Composer { return &subscription{} })
	// business only
	app.Route("/plan", func() app.Composer { return &plan{} })
	app.Route("/associates", func() app.Composer { return &associate{} })
	app.Route("/clients", func() app.Composer { return &client{} })
	app.Route("/suppliers", func() app.Composer { return &supplier{} })
	app.Route("/terms", func() app.Composer { return &terms{} })
	app.Route("/privacy", func() app.Composer { return &privacy{} })
	app.Route("/cookie", func() app.Composer { return &cookie{} })

	// Once the routes set up, the next thing to do is to either launch the app
	// or the server that serves the app.
	//
	// When executed on the client-side, the RunWhenOnBrowser() function
	// launches the app,  starting a loop that listens for app events and
	// executes client instructions. Since it is a blocking call, the code below
	// it will never be executed.
	//
	// When executed on the server-side, RunWhenOnBrowser() does nothing, which
	// lets room for server implementation without the need for precompiling
	// instructions.
	app.RunWhenOnBrowser()

	// Finally, launching the server that serves the app is done by using the Go
	// standard HTTP package.
	//
	// The Handler is an HTTP handler that serves the client and all its
	// required resources to make it work into a web browser. Here it is
	// configured to handle requests with a path that starts with "/".
	http.Handle("/", &app.Handler{
		Name:        "Cyber GUBI",
		Description: "An unconditional universal basic income",
		Styles: []string{
			"/web/app.css", // Loads app.css file.
		},
		Scripts: []string{
			"web/script.js",
			"web/helpers.js",
		},
		RawHeaders: []string{
			`
			<script src="https://cdn.jsdelivr.net/npm/@vladmandic/face-api/dist/face-api.js"></script>`,
		},
	})

	http.Handle("/auth", &app.Handler{
		Name:        "Cyber GUBI",
		Description: "An unconditional universal basic income",
		Styles: []string{
			"/web/app.css", // Loads app.css file.
		},
		Scripts: []string{
			"web/script.js",
			"web/helpers.js",
		},
		RawHeaders: []string{
			`
			<script src="https://cdn.jsdelivr.net/npm/@vladmandic/face-api/dist/face-api.js"></script>`,
		},
	})

	http.Handle("/wallet", &app.Handler{
		Name:        "Cyber GUBI",
		Description: "An unconditional universal basic income",
		Styles: []string{
			"/web/app.css", // Loads app.css file.
		},
	})

	http.Handle("/wallets", &app.Handler{
		Name:        "Cyber GUBI",
		Description: "An unconditional universal basic income",
		Styles: []string{
			"/web/app.css", // Loads app.css file.
		},
		Scripts: []string{
			"web/helpers.js",
		},
	})

	http.Handle("/transactions", &app.Handler{
		Name:        "Cyber GUBI",
		Description: "An unconditional universal basic income",
		Styles: []string{
			"/web/app.css", // Loads app.css file.
		},
		Scripts: []string{
			"web/helpers.js",
		},
	})

	http.Handle("/tax", &app.Handler{
		Name:        "Cyber GUBI",
		Description: "An unconditional universal basic income",
		Styles: []string{
			"/web/app.css", // Loads app.css file.
		},
	})

	http.Handle("/payment", &app.Handler{
		Name:        "Cyber GUBI",
		Description: "An unconditional universal basic income",
		Styles: []string{
			"/web/app.css", // Loads app.css file.
		},
	})

	http.Handle("/subscriptions", &app.Handler{
		Name:        "Cyber GUBI",
		Description: "An unconditional universal basic income",
		Styles: []string{
			"/web/app.css", // Loads app.css file.
		},
	})

	http.Handle("/plan", &app.Handler{
		Name:        "Cyber GUBI",
		Description: "An unconditional universal basic income",
		Styles: []string{
			"/web/app.css", // Loads app.css file.
		},
	})

	http.Handle("/associates", &app.Handler{
		Name:        "Cyber GUBI",
		Description: "An unconditional universal basic income",
		Styles: []string{
			"/web/app.css", // Loads app.css file.
		},
	})

	http.Handle("/clients", &app.Handler{
		Name:        "Cyber GUBI",
		Description: "An unconditional universal basic income",
		Styles: []string{
			"/web/app.css", // Loads app.css file.
		},
	})

	http.Handle("/suppliers", &app.Handler{
		Name:        "Cyber GUBI",
		Description: "An unconditional universal basic income",
		Styles: []string{
			"/web/app.css", // Loads app.css file.
		},
	})

	http.Handle("/terms", &app.Handler{
		Name:        "Cyber GUBI",
		Description: "An unconditional universal basic income",
		Styles: []string{
			"/web/app.css", // Loads app.css file.
		},
	})

	http.Handle("/privacy", &app.Handler{
		Name:        "Cyber GUBI",
		Description: "An unconditional universal basic income",
		Styles: []string{
			"/web/app.css", // Loads app.css file.
		},
	})

	http.Handle("/cookie", &app.Handler{
		Name:        "Cyber GUBI",
		Description: "An unconditional universal basic income",
		Styles: []string{
			"/web/app.css", // Loads app.css file.
		},
	})

	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal(err)
	}
}
