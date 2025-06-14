package main

import (
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type termsBusiness struct {
	app.Compo
}

// The Render method is where the component appearance is defined. Here, a
// webauthn is displayed.
func (t *termsBusiness) Render() app.UI {
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
							app.Span().Class("span-docs").Text("1. Cyber-gubi is a p2p platform where you can offer services and products."),
							app.Span().Class("span-docs").Text("2. You can only have one active subscription all-in-one plan at any given time."),
							app.Span().Class("span-docs").Text("3. Taxes are collected by your government without your involvement."),
							app.Span().Class("span-docs").Text("4. If you want to dispute collected tax you have to do it on your own outside of the platform."),
							app.Span().Class("span-docs").Text("5. Cyber-gubi is not owned by anyone and doesn't carry any liability or responsibility itself."),
							app.Span().Class("span-docs").Text("6. You are responsible for providing a valid VAT number."),
							app.Span().Class("span-docs").Text("7. It's up to you to set prices based on free-market economy."),
							app.Span().Class("span-docs").Text("8. The arbitrage between prices and wages is solely set by you."),
							app.Span().Class("span-docs").Text("9. Your biometrics data is yours."),
							app.Span().Class("span-docs").Text("10. It is stored as hashed public data on IPFS."),
							app.Span().Class("span-docs").Text("11. All content stored on the platform is public."),
						),
					),
				),
			),
		),
	)
}
