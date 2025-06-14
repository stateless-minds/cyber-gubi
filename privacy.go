package main

import (
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type privacy struct {
	app.Compo
}

// The Render method is where the component appearance is defined. Here, a
// webauthn is displayed.
func (p *privacy) Render() app.UI {
	return app.Div().Class("container").Body(
		app.Div().Class("mobile").Body(
			app.Div().Class("header").Body(
				newNav(),
				app.Div().Class("header-summary").Body(
					app.Span().Class("logo").Text("cyber-gubi"),
					app.Div().Class("summary-text").Body(
						app.Span().Text("Authentication"),
					),
				),
			),
			app.Div().ID("content").Body(
				app.Div().Class("card").Body(
					app.Div().Class("upper-row").Body(
						app.Div().Class("card-item card-terms").Body(
							app.Span().Class("span-header").Text("Privacy Policy"),
							app.Span().Class("span-docs").Text("1. Cyber-gubi is GDPR compliant and uses your biometrics data to identify you and to ensure one wallet per person."),
							app.Span().Class("span-docs").Text("2. Your biometrics data is yours."),
							app.Span().Class("span-docs").Text("4. It is stored as hashed public data on IPFS."),
							app.Span().Class("span-docs").Text("5. You can delete your account and biometrics data anytime."),
							app.Span().Class("span-docs").Text("6. All user content generated on the platform is public and not owned by anyone."),
						),
					),
				),
			),
		),
	)
}
