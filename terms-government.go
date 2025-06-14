package main

import (
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type termsGovernment struct {
	app.Compo
}

// The Render method is where the component appearance is defined. Here, a
// webauthn is displayed.
func (t *termsGovernment) Render() app.UI {
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
							app.Span().Class("span-header").Text("Terms of Use"),
							app.Span().Class("span-docs").Text("In order to use cyber-gubi you agree to the below terms."),
							app.Span().Class("span-docs").Text("1. Cyber-gubi is a p2p platform where you can collect taxes from citizens and businesses."),
							app.Span().Class("span-docs").Text("2. In order to register you need to provide a form of government ID which can be cross-checked in a public database and your biometric data."),
							app.Span().Class("span-docs").Text("3. You can later add and remove associates with their biometric data."),
							app.Span().Class("span-docs").Text("4. Calculating and collecting tax is your sole responsibility."),
							app.Span().Class("span-docs").Text("5. Cyber-gubi is not owned by anyone and doesn't carry any liability or responsibility itself."),
							app.Span().Class("span-docs").Text("6. You have no control over the platform and can not exchange cyber-gubi for any other currency."),
							app.Span().Class("span-docs").Text("7. You can not regulate the market in any other way except taxes."),
							app.Span().Class("span-docs").Text("8. All data on the platform is public and visibile to you in your jurisdiction."),
							app.Span().Class("span-docs").Text("9. Your biometrics data is yours."),
							app.Span().Class("span-docs").Text("10. It is stored as hashed public data on IPFS."),
						),
					),
				),
			),
		),
	)
}
