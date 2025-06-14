package main

import (
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type terms struct {
	app.Compo
}

// The Render method is where the component appearance is defined. Here, a
// webauthn is displayed.
func (t *terms) Render() app.UI {
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
							app.Span().Class("span-docs").Text("1. Cyber-gubi is a digital wallet for guaranteed basic income."),
							app.Span().Class("span-docs").Text("2. Everyone is eligible to receive basic income with no exceptions and conditions."),
							app.Span().Class("span-docs").Text("3. It is a peer-to-peer open-source app not owned by anyone and hosted by everyone."),
							app.Span().Class("span-docs").Text("4. A real-time inflation indexer adjusts the income based on real-time analysis of price fluctuations."),
							app.Span().Class("span-docs").Text("5. The indexer is distributed and runs on each device the last 3 days of the month."),
							app.Span().Class("span-docs").Text("6. It automatically adjusts the new basic income amount for next month."),
							app.Span().Class("span-docs").Text("7. The basic income is automatically received when you login in any of the last 3 days of each month."),
							app.Span().Class("span-docs").Text("8. Cyber-gubi is GDPR compliant and uses your biometrics data to identify you and to ensure one wallet per person."),
							app.Span().Class("span-docs").Text("9. Your biometrics data is yours."),
							app.Span().Class("span-docs").Text("10. It is stored as hashed public data on IPFS."),
							app.Span().Class("span-docs").Text("11. All user content generated on the platform is public and not owned by anyone."),
						),
					),
				),
			),
		),
	)
}
