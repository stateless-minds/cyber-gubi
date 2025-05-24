package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	shell "github.com/stateless-minds/go-ipfs-api"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

const dbUser = "user"
const dbInflation = "inflation"

// auth is a component that uses webauthn and biometrics. A component is a
// customizable, independent, and reusable UI element. It is created by
// embedding app.Compo into a struct.
type auth struct {
	app.Compo
	sh                     *shell.Shell
	webAuthn               *webauthn.WebAuthn
	descriptorJSON         string
	userID                 string
	credentialID           string
	currentUser            User
	countryCode            string
	region                 string
	entity                 string
	termsAccepted          bool
	notificationPermission app.NotificationPermission
	businessName           string
	associateName          string
	newAssociateName       string
	businessID             string
	isCountry              bool
}

// Credential represents the structure for credential information.
type Credential struct {
	ID            []byte        `mapstructure:"id" json:"id"`
	PublicKey     []byte        `mapstructure:"publicKey" json:"public_key"`
	Authenticator Authenticator `mapstructure:"authenticator" json:"authenticator"`
}

// Authenticator represents the authenticator details.
type Authenticator struct {
	AAGUID       []byte `mapstructure:"AAGUID" json:"aaguid"`
	Attachment   string `mapstructure:"attachment" json:"attachment"`
	CloneWarning bool   `mapstructure:"cloneWarning" json:"clone_warning"`
	SignCount    int    `mapstructure:"signCount" json:"sign_count"`
}

type User struct {
	ID            []byte                `mapstructure:"_id" json:"_id" validate:"uuid_rfc4122"`                       // Unique identifier for the user (should be a byte array)
	Name          string                `mapstructure:"name" json:"name" validate:"uuid_rfc4122"`                     // Username or identifier for the user
	DisplayName   string                `mapstructure:"display_name" json:"display_name" validate:"uuid_rfc4122"`     // Display name for the user
	CredentialIDs []webauthn.Credential `mapstructure:"credential_ids" json:"credential_ids" validate:"uuid_rfc4122"` // List of credential IDs associated with the user
	Descriptor    json.RawMessage       `mapstructure:"descriptor" json:"descriptor" validate:"uuid_rfc4122"`         // Face descriptor for the user
	BusinessID    string                `mapstructure:"business_id" json:"business_id" validate:"uuid_rfc4122"`       // Company business ID
	GovernmentID  string                `mapstructure:"government_id" json:"government_id" validate:"uuid_rfc4122"`   // Government ID
	CountryCode   string                `mapstructure:"country_code" json:"country_code" validate:"uuid_rfc4122"`
	Region        string                `mapstructure:"region" json:"region" validate:"uuid_rfc4122"` // Country
}

// Define your own struct that matches the CredentialCreation structure
type MyCredentialCreation struct {
	Challenge []byte
	RP        RelyingParty
	User      User
}

type RelyingParty struct {
	Name string
	ID   string
}

type UserVerification struct {
	UserVerificationRequirement string
}

// PublicKeyCredentialType represents the type of public key credential.
type PublicKeyCredentialType string

const (
	PublicKeyCredentialTypePublicKey PublicKeyCredentialType = "public-key"
)

// Parameters represents the parameters for public key credentials.
type Parameters struct {
	Type PublicKeyCredentialType `json:"type"` // Type of credential (e.g., "public-key")
	Alg  int                     `json:"alg"`  // COSE algorithm identifier
}

// RegistrationData represents the structure of the registration data
type RegistrationData struct {
	ID       string         `json:"id"`
	RawID    string         `json:"rawId"`
	Type     string         `json:"type"`
	Response ResponseCreate `json:"response"`
}

// RegistrationData represents the structure of the registration data
type LoginData struct {
	ID       string      `json:"id"`
	RawID    string      `json:"rawId"`
	Type     string      `json:"type"`
	Response ResponseGet `json:"response"`
}

type ResponseCreate struct {
	ClientDataJSON    string `json:"clientDataJSON"`
	AttestationObject string `json:"attestationObject"`
}

type ResponseGet struct {
	ClientDataJSON    string `json:"clientDataJSON"`
	AuthenticatorData string `json:"authenticatorData"`
	Signature         string `json:"signature"`
}

// Implementing the webauthn.User interface
func (u *User) WebAuthnID() []byte {
	return u.ID
}

func (u *User) WebAuthnName() string {
	return u.Name
}

func (u *User) WebAuthnDisplayName() string {
	return u.DisplayName
}

func (u *User) WebAuthnCredentials() []webauthn.Credential {
	return u.CredentialIDs
}

// Methods to manage credentials
func (u *User) AddCredential(credential webauthn.Credential) {
	u.CredentialIDs = append(u.CredentialIDs, credential)
}

func (u *User) UpdateCredential(credential webauthn.Credential) {
	for i, cred := range u.CredentialIDs {
		if string(cred.ID) == string(credential.ID) {
			u.CredentialIDs[i] = credential // Update existing credential ID
			break
		}
	}
}

func (a *auth) OnMount(ctx app.Context) {
	a.notificationPermission = ctx.Notifications().Permission()
	if a.notificationPermission == "default" {
		a.notificationPermission = ctx.Notifications().RequestPermission()
	}

	sh := shell.NewShell("localhost:5001")
	a.sh = sh

	wconfig := &webauthn.Config{
		RPDisplayName: "cyber-gubi",                      // Display Name for your site
		RPID:          "localhost",                       // Generally the FQDN for your site
		RPOrigins:     []string{"http://localhost:8000"}, // Allowed origins for WebAuthn requests
	}

	var err error

	if a.webAuthn, err = webauthn.New(wconfig); err != nil {
		ctx.Notifications().New(app.Notification{
			Title: "Webauthn instantiate error",
			Body:  err.Error(),
		})
		log.Fatal(err)
	}

	ctx.Async(func() {
		a.fetchDescriptor()
		ctx.Dispatch(func(ctx app.Context) {
			days := daysRemainingInMonth(time.Now())
			if days <= 3 {
				a.getIncome(ctx)
			}
		})
	})

	// a.isCountry = true
	// a.beginRegistration(ctx)

	// a.getUsers()
	// a.deleteUsers()
	// a.getCountryWallets()
	// a.deleteWallets()

	// err = a.sh.CreateCountryAccounts()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// err = a.sh.UpdateCountryAccount("Bulgaria", "123456789", "[-0.0627824,0.0557961,0.0648634,0.0306014,-0.1085096,-0.0788037,0.075048,-0.1441734,0.1718586,-0.0523681,0.2802132,-0.0567889,-0.1854445,-0.0597611,0.0114786,0.0480673,-0.0496276,-0.1171764,-0.0804037,-0.0205053,0.0530239,0.0250173,0.0605471,0.0514089,-0.1221177,-0.3250661,-0.0808559,-0.0847882,0.0657298,-0.0954366,0.0462636,0.1540155,-0.1214376,-0.1278621,0.0442731,0.0386029,-0.0235876,-0.0453274,0.2145044,-0.0412263,-0.1723431,0.030252,-0.0042006,0.3438928,0.166643,-0.0065849,0.067531,-0.1344344,0.072491,-0.2080505,0.1440588,0.0992688,0.0965577,0.0192163,0.1379771,-0.1077227,0.0233076,0.0733253,-0.2029213,0.1296015,0.1562316,-0.0187693,0.0118775,-0.0894548,0.1507326,0.0448835,-0.065876,-0.0924555,0.0696441,-0.1390269,-0.0671479,0.1045424,-0.1480977,-0.2013738,-0.2597257,0.060221,0.4447426,0.157009,-0.2232685,0.0326051,-0.0309981,0.007613,0.1382988,0.0460388,-0.0278781,-0.0480645,-0.1229947,0.0531989,0.1622549,0.0077055,-0.1267886,0.2304303,-0.0159248,-0.0801122,0.0890828,0.0514886,-0.0796351,0.0404276,-0.1380173,-0.0350928,0.0846586,-0.0621224,0.0212492,0.0748239,-0.1815445,0.1994248,-0.0428519,0.0419655,0.037446,-0.0585382,-0.108566,0.0453554,0.1541657,-0.2565612,0.1867651,0.0518779,0.0431454,0.1520243,0.0069921,0.0518195,0.010262,0.0022844,-0.0952344,-0.0429271,0.0531778,0.0003472,0.1194178,0.0360485]")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// err = a.sh.CreateCountryWallets()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// w.deleteIncome()
	// w.deleteTransactions()
	// w.deleteInflation()
	// w.deletePlans()
	// w.deleteSubscriptions()
	// return

	ctx.ObserveState("entity", &a.entity)

	ctx.ObserveState("termsAccepted", &a.termsAccepted).
		OnChange(func() {
			if a.entity == "individual" {
				a.findCountry(ctx)
				a.beginRegistration(ctx)
			}
		})

	ctx.ObserveState("businessID", &a.businessID).
		OnChange(func() {
			log.Println("a.businessID: ", a.businessID)
			if a.entity == "business" && a.termsAccepted {
				ctx.GetState("businessName", &a.businessName)
				ctx.GetState("associateName", &a.associateName)

				a.findCountry(ctx)
				duplicateFound := a.isDuplicate(ctx)
				if !duplicateFound {
					a.beginRegistration(ctx)
				}
			}
		})
}

func (w *wallet) deleteIncome() {
	err := w.sh.OrbitDocsDelete(dbIncome, "all")
	if err != nil {
		log.Fatal(err)
	}
}

func (w *wallet) deleteTransactions() {
	err := w.sh.OrbitDocsDelete(dbTransaction, "all")
	if err != nil {
		log.Fatal(err)
	}
}

func (w *wallet) deleteInflation() {
	err := w.sh.OrbitDocsDelete(dbInflation, "all")
	if err != nil {
		log.Fatal(err)
	}
}

func (w *wallet) deletePlans() {
	err := w.sh.OrbitDocsDelete(dbPlan, "all")
	if err != nil {
		log.Fatal(err)
	}
}

func (w *wallet) deleteSubscriptions() {
	err := w.sh.OrbitDocsDelete(dbSubscription, "all")
	if err != nil {
		log.Fatal(err)
	}
}

func (a *auth) getCountryWallets() {
	_, err := a.sh.OrbitDocsQuery(dbWallet, "all", "")
	if err != nil {
		log.Fatal(err)
	}
}

func (a *auth) getUsers() {
	_, err := a.sh.OrbitDocsQuery(dbUser, "all", "")
	if err != nil {
		log.Fatal(err)
	}
}

func (a *auth) findCountry(ctx app.Context) {
	myPeer, err := a.sh.ID()
	if err != nil {
		log.Fatal(err)
	}

	for _, addr := range myPeer.Addresses {
		// Split the address to handle multiaddr format
		ip, err := extractIP(addr)
		if err != nil {
			continue
		}
		if strings.Contains(ip, ":") {
			continue
		}
		if isPublicIP(ip) {
			fmt.Println("Potential public IP:", ip)
			ctx.Async(func() {
				r, err := http.Get("http://ip-api.com/json/" + ip + "?fields=countryCode,region")
				if err != nil {
					log.Fatal(err)
				}
				defer r.Body.Close()

				b, err := ioutil.ReadAll(r.Body)
				if err != nil {
					log.Fatal(err)
				}

				var info map[string]interface{}

				err = json.Unmarshal(b, &info)
				if err != nil {
					log.Fatal(err)
				}

				// Storing HTTP response in component field:
				ctx.Dispatch(func(ctx app.Context) {
					a.countryCode = info["countryCode"].(string)
					ctx.SetState("countryCode: ", a.countryCode)

					a.region = info["region"].(string)
					ctx.SetState("region: ", a.region)
				})
			})
		}
	}
}

func isPublicIP(ip string) bool {
	ipAddr := net.ParseIP(ip)
	if ipAddr != nil {
		// Check if the IP is not a private or loopback address
		if ipAddr.IsPrivate() || ipAddr.IsLoopback() {
			return false
		}
		return true
	}
	return false
}

func extractIP(addr string) (string, error) {
	// Simple function to extract IP from multiaddr format
	// This might need adjustments based on the actual format of addr
	parts := strings.Split(addr, "/")
	for _, part := range parts {
		if net.ParseIP(part) != nil {
			return part, nil
		}
	}
	return "", fmt.Errorf("no IP found in address")
}

func (a *auth) getIncome(ctx app.Context) {
	ctx.Async(func() {
		i, err := a.sh.OrbitDocsQuery(dbIncome, "all", "")
		if err != nil {
			log.Fatal(err)
		}

		income := []Income{}

		if len(i) == 0 {
			log.Fatal(err)
		}

		err = json.Unmarshal([]byte(i), &income) // Unmarshal the byte slice directly
		if err != nil {
			log.Fatal(err)
		}

		ctx.Dispatch(func(ctx app.Context) {
			doIndexer := true
			for _, inc := range income {
				if inc.Period == strconv.Itoa(time.Now().Year())+"/"+strconv.Itoa(int(time.Now().Month()+1)) {
					doIndexer = false
				}
			}

			if doIndexer {
				a.sh.RunInflationIndexer()
			}
		})
	})
}

// Function to generate a new user
func NewUser() (*User, error) {
	return &User{
		ID:            protocol.URLEncodedBase64(uuid.NewString()),
		CredentialIDs: []webauthn.Credential{}, // Initialize with no credentials
	}, nil
}

// Generate random hyperplanes for LSH
func generateHyperplanes(dim, numHashes int) [][]float64 {
	rand.Seed(time.Now().UnixNano())
	hyperplanes := make([][]float64, numHashes)
	for i := 0; i < numHashes; i++ {
		hp := make([]float64, dim)
		for j := 0; j < dim; j++ {
			hp[j] = rand.NormFloat64() // Gaussian random values
		}
		hyperplanes[i] = hp
	}
	return hyperplanes
}

// Compute binary LSH hash signature from descriptor vector and hyperplanes
func computeLSHHash(descriptor []float64, hyperplanes [][]float64) []bool {
	signature := make([]bool, len(hyperplanes))
	for i, hp := range hyperplanes {
		dot := 0.0
		for j, val := range descriptor {
			dot += val * hp[j]
		}
		signature[i] = dot >= 0
	}
	return signature
}

// Convert bool slice to string of '0' and '1'
func boolSliceToString(bits []bool) string {
	s := make([]byte, len(bits))
	for i, bit := range bits {
		if bit {
			s[i] = '1'
		} else {
			s[i] = '0'
		}
	}
	return string(s)
}

// Optionally hash the binary string to get a fixed-length bucket key
func hashBand(band string) string {
	h := sha256.Sum256([]byte(band))
	return hex.EncodeToString(h[:])
}

func uniqueStrings(input []string) []string {
	m := make(map[string]struct{})
	var result []string
	for _, s := range input {
		if _, exists := m[s]; !exists {
			m[s] = struct{}{}
			result = append(result, s)
		}
	}
	return result
}

func (a *auth) doCheck(ctx app.Context, e app.Event) {
	a.descriptorJSON = e.Get("detail").Get("descriptor").String()
	if len(a.descriptorJSON) == 0 {
		log.Fatal("descriptorJSON is empty")
	}
	ctx.GetState("newAssociateName", &a.newAssociateName)
	if len(a.newAssociateName) > 0 {
		a.updateUser(ctx)
	} else {
		user, err := a.sh.OrbitDocsQuery(dbUser, "descriptor", a.descriptorJSON)
		if err != nil {
			log.Fatal(err)
		}

		if len(user) == 0 {
			// register
			app.Window().GetElementByID("main-menu").Call("click")
		} else {
			// login
			var u User

			err := json.Unmarshal(user, &u)
			if err != nil {
				log.Fatal(err)
			}

			ctx.SetState("countryCode", u.CountryCode)

			currentUser := u

			// business waitlist
			if len(currentUser.BusinessID) > 0 {
				ctx.Notifications().New(app.Notification{
					Title: "Check back later",
					Body:  "Cyber-gubi is not yet open to businesses and you are in the waitlist. Check https://github.com/stateless-minds/cyber-gubi for an announcement.",
				})
				return
			}

			var descriptorFloat []float64
			descriptorLSH := make(map[int][]string)
			matches := make(map[string]int)

			err = json.Unmarshal([]byte(a.descriptorJSON), &descriptorFloat)
			if err != nil {
				log.Fatal(err)
			}

			// Generate 32 random hyperplanes for LSH
			hyperplanes := generateHyperplanes(128, 32)

			// Compute binary LSH signature
			signature := computeLSHHash(descriptorFloat, hyperplanes)

			// Split signature into 8 bands of 4 bits each and hash each band
			bands := 8
			rowsPerBand := len(signature) / bands
			for b := range bands {
				bandBits := signature[b*rowsPerBand : (b+1)*rowsPerBand]
				bandStr := boolSliceToString(bandBits)
				bucketKey := hashBand(bandStr)
				descriptorLSH[b] = append(descriptorLSH[b], bucketKey)
				descriptorLSH[b] = uniqueStrings(descriptorLSH[b])
			}

			log.Println(descriptorLSH)

			ctx.SetState("userID", string(currentUser.ID))
			if len(currentUser.BusinessID) > 0 || len(currentUser.GovernmentID) > 0 {
				ctx.SetState("currentUser", currentUser).Persist()

				var descriptorHash map[string]map[int][]string

				err := json.Unmarshal(currentUser.Descriptor, &descriptorHash)
				if err != nil {
					log.Fatal(err)
				}

				for name, desc := range descriptorHash {
					for _, d := range desc {
						for _, v := range descriptorLSH {
							for _, b := range d {
								for _, f := range v {
									if b == f {
										matches[name]++
									}
								}
							}
						}
					}
				}

				log.Println("matches: ", matches)

				if len(matches) > 0 {
					var maxKey string
					var maxValue int
					first := true

					for k, v := range matches {
						if v == maxValue {
							ctx.Notifications().New(app.Notification{
								Title: "Error",
								Body:  "Face not recognized. Trying again.",
							})
							ctx.Reload()
						}
						if first || v > maxValue {
							maxValue = v
							maxKey = k
							first = false
						}
					}

					ctx.SetState("associateName", maxKey).Persist()

					if len(currentUser.BusinessID) > 0 {
						ctx.SetState("isBusiness", true)
						ctx.SetState("businessName", currentUser.DisplayName)
					} else {
						ctx.SetState("isGovernment", true)
						ctx.SetState("countryName", currentUser.DisplayName)
					}
					a.credentialID = string(currentUser.CredentialIDs[0].ID)
				} else {
					ctx.Reload()
				}
			}

			a.beginLogin(ctx)
		}
	}
}

func daysRemainingInMonth(date time.Time) int {
	// Calculate the first day of the next month
	firstDayOfNextMonth := time.Date(date.Year(), date.Month()+1, 1, 0, 0, 0, 0, date.Location())

	// Subtract one day to get the last day of the current month
	lastDayOfMonth := firstDayOfNextMonth.Add(-time.Hour * 24)

	// Set the current date to midnight
	midnightToday := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	// Calculate the difference between the last day of the month and midnight today
	diff := lastDayOfMonth.Sub(midnightToday)

	// Convert the duration to days
	days := int(diff.Hours()/24) + 1 // Add 1 to include today

	return days
}

func (a *auth) fetchDescriptor() {
	// Send response event to child
	app.Window().Get("parent").Get("window").Call("dispatchEvent", // Target the iframe's window
		app.Window().Get("CustomEvent").New("fetchDescriptor"),
	)
}

func (a *auth) deleteUsers() {
	err := a.sh.OrbitDocsDelete(dbUser, "all")
	if err != nil {
		log.Fatal(err)
	}
}

func (a *auth) deleteWallets() {
	err := a.sh.OrbitDocsDelete(dbWallet, "all")
	if err != nil {
		log.Fatal(err)
	}
}

func (a *auth) createUser(ctx app.Context) {
	ctx.Async(func() {
		var descriptor []float64
		err := json.Unmarshal([]byte(a.descriptorJSON), &descriptor)
		if err != nil {
			log.Fatal(err)
		}

		descriptorMap := make(map[string][]float64)
		if len(a.associateName) == 0 {
			// pseudonymous for individuals
			descriptorMap["user"] = descriptor
		} else {
			descriptorMap[a.associateName] = descriptor
		}

		dm, err := json.Marshal(descriptorMap)
		if err != nil {
			log.Fatal(err)
		}

		user := User{
			Name:        a.businessName,
			DisplayName: a.businessName,
			ID:          protocol.URLEncodedBase64(a.userID),
			CredentialIDs: []webauthn.Credential{
				{
					ID: []byte(a.credentialID),
				},
			},
			Descriptor:  dm,
			BusinessID:  a.businessID,
			CountryCode: a.countryCode,
			Region:      a.region,
		}

		userJSON, err := json.Marshal(user)
		if err != nil {
			log.Fatal(err)
		}

		err = a.sh.OrbitDocsPut(dbUser, userJSON)
		if err != nil {
			log.Println(err.Error())
			if err.Error() == "orbit/docsput: duplicate found" {
				ctx.Notifications().New(app.Notification{
					Title: "Registration error",
					Body:  "Duplicate foun. Try a different name.",
				})
			} else {
				ctx.Notifications().New(app.Notification{
					Title: "Registration error",
					Body:  "Oops something unexpected happened. Try again or open an issue here: https://github.com/stateless-minds/cyber-gubi/issues",
				})
			}
			return
		}

		ctx.Dispatch(func(ctx app.Context) {
			a.currentUser = user
			if len(a.currentUser.BusinessID) > 0 {
				// waitlist
				ctx.Notifications().New(app.Notification{
					Title: "Check back later",
					Body:  "Cyber-gubi is not yet open to businesses and you are in the waitlist. Check https://github.com/stateless-minds/cyber-gubi for an announcement.",
				})
				return
				// ctx.SetState("currentUser", a.currentUser).Persist()
				// ctx.SetState("isBusiness", true)
			} else {
				a.beginLogin(ctx)
			}
		})
	})
}

func (a *auth) updateUser(ctx app.Context) {
	ctx.Async(func() {
		var descriptor []float64
		err := json.Unmarshal([]byte(a.descriptorJSON), &descriptor)
		if err != nil {
			log.Fatal(err)
		}

		ctx.GetState("currentUser", &a.currentUser)

		var descriptorHashes map[string]interface{}

		err = json.Unmarshal(a.currentUser.Descriptor, &descriptorHashes)
		if err != nil {
			log.Fatal(err)
		}

		descriptorHashes[a.newAssociateName] = descriptor

		a.currentUser.Descriptor, err = json.Marshal(descriptorHashes)
		if err != nil {
			log.Fatal(err)
		}

		userJSON, err := json.Marshal(a.currentUser)
		if err != nil {
			log.Fatal(err)
		}

		err = a.sh.OrbitDocsPut(dbUser, userJSON)
		if err != nil {
			log.Fatal(err)
		}

		ctx.Dispatch(func(ctx app.Context) {
			ctx.DelState("newAssociateName")
			ctx.DelState("currentUser")
			ctx.DelState("associateName")
			ctx.Notifications().New(app.Notification{
				Title: "Success",
				Body:  "Associate " + a.newAssociateName + " has been added. Any of you can log in now. To use another device simply copy keys across.",
			})
			ctx.Reload()
		})
	})
}

func (a *auth) isDuplicate(ctx app.Context) bool {
	res, err := a.sh.OrbitDocsQuery(dbUser, "all", "")
	if err != nil {
		log.Fatal(err)
	}

	users := []map[string]interface{}{}

	if len(res) != 0 {
		err = json.Unmarshal([]byte(res), &users)
		if err != nil {
			log.Fatal(err)
		}
	}

	duplicateName := false
	duplicateBusinessID := false
	sameCountry := false

	if len(users) > 0 {
		for _, usrs := range users {
			for k, v := range usrs {
				if k == "name" || k == "display_name" {
					if v == a.businessName {
						duplicateName = true
					}
				} else if k == "business_id" {
					if v == a.businessID {
						duplicateBusinessID = true
					}
				} else if k == "country_code" {
					sameCountry = true
				}
			}
		}
	}

	if sameCountry && duplicateName {
		ctx.Notifications().New(app.Notification{
			Title: "Registration error",
			Body:  "Business with this name already exists.",
		})
		return true
	} else if sameCountry && duplicateBusinessID {
		ctx.Notifications().New(app.Notification{
			Title: "Registration error",
			Body:  "Business with this ID already exists in your country.",
		})
		return true
	}

	return false
}

func (a *auth) beginRegistration(ctx app.Context) {
	// RelyingParty instance
	relyingParty := RelyingParty{
		Name: a.webAuthn.Config.RPDisplayName,
		ID:   a.webAuthn.Config.RPID,
	}

	userID := uuid.NewString()

	us := User{
		ID: []byte(userID),
	}

	if len(a.businessName) > 0 {
		us.Name = a.businessName
		us.DisplayName = a.businessName
	}

	rp := app.ValueOf(map[string]interface{}{
		"name": relyingParty.Name,
		"id":   relyingParty.ID,
	})

	usr := app.ValueOf(map[string]interface{}{
		"name":        us.Name,
		"displayName": us.DisplayName,
	})

	usID := app.Window().Get("Uint8Array").New(len(us.ID))

	usr.Set("id", usID)

	as := app.ValueOf(map[string]interface{}{
		"authenticatorAttachment": "platform",
		"userVerification":        "required",
		"residentKey":             "required",
	})

	// Create pubKeyCredParams as an array in JavaScript
	pubKeyCredParams := app.Window().Get("Array").New()

	// Add parameters to the pubKeyCredParams array
	param1 := app.Window().Get("Object").New()
	param1.Set("type", "public-key")
	param1.Set("alg", -7) // Example algorithm identifier for ES256
	pubKeyCredParams.Call("push", param1)

	param2 := app.Window().Get("Object").New()
	param2.Set("type", "public-key")
	param2.Set("alg", -257) // Example algorithm identifier for RS256
	pubKeyCredParams.Call("push", param2)

	// Generate a random challenge
	// Create a new random source seeded with the current time
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	challengeByteArray := make([]byte, 32)
	r.Read(challengeByteArray)
	challenge := app.Window().Get("Uint8Array").New(len(challengeByteArray)) // Generate a random challenge
	// Fill the Uint8Array with the byte values
	for i, b := range challengeByteArray {
		challenge.SetIndex(i, b)
	}

	obj := app.Window().Get("Object").New()
	obj.Set("challenge", challenge)
	obj.Set("rp", rp)
	obj.Set("user", usr)
	obj.Set("pubKeyCredParams", pubKeyCredParams)
	obj.Set("authenticatorSelection", as)
	obj.Set("publicKey", obj)

	// Access the navigator object
	promise := app.Window().Get("navigator").Get("credentials").Call("create", obj)

	// Step 3: Handle the promise response
	promise.Call("then", app.FuncOf(func(this app.Value, args []app.Value) interface{} {
		if len(args) > 0 {
			cred := args[0] // The PublicKeyCredential object
			// Get the credentialId
			credentialID := cred.Get("id").String()
			a.userID = userID
			a.credentialID = credentialID
			ctx.SetState("userID", userID)
			if len(a.businessID) > 0 {
				ctx.SetState("isBusiness", true)
			}
			if a.isCountry {
				a.sh.UpdateCountryAccount("Germany", "123456789", "Zigmund Paprikashliev", "[-0.0673568,0.0234374,0.0802006,-0.0631915,-0.0686559,-0.0847427,-0.0515263,-0.1155665,0.1495542,-0.1037179,0.1790959,-0.0015682,-0.2226413,-0.050798,-0.0436366,0.120865,-0.1862528,-0.0986364,-0.0070285,-0.0205311,0.1345131,0.1120488,0.1475301,0.1279192,-0.2086958,-0.302754,-0.1117638,-0.0752963,-0.0156373,-0.0942581,0.010295,0.1110142,-0.1910131,0.0031465,0.0200539,0.1487437,-0.0096367,-0.0923902,0.1378625,0.027024,-0.2523024,-0.0762603,0.0163836,0.3076439,0.1521883,-0.0481748,0.0311912,-0.0459495,0.0724058,-0.2650769,-0.0102193,0.153466,-0.0181576,0.012071,0.0954404,-0.1148628,0.0752044,0.0939575,-0.1404806,0.0166856,0.0178389,-0.1034788,0.0485206,-0.0392122,0.1182094,0.0230354,-0.1008821,-0.1097133,0.0994171,-0.1891849,-0.0356221,0.1585451,-0.1541384,-0.1974083,-0.3089304,-0.0857801,0.3778991,0.075122,-0.1312303,0.094451,-0.0267344,0.020191,0.0687223,0.1576998,0.0024204,0.1569585,-0.1685978,0.130646,0.1625986,-0.0988349,0.0168352,0.2774154,0.013526,-0.0127363,0.112529,0.1045961,-0.1005734,0.0173907,-0.1334615,0.0534191,0.0694218,-0.073896,-0.0183913,0.0993738,-0.1585376,0.1131142,0.0016718,-0.0896451,-0.0046888,-0.0411574,-0.1226987,-0.1067363,0.1192402,-0.2494856,0.087643,0.2089535,-0.0419069,0.1515342,-0.0348665,0.1065585,-0.0879387,-0.0429889,-0.0769198,-0.0358359,0.0283674,-0.0343756,-0.0201668,-0.0680671]", a.credentialID)
			} else {
				a.createUser(ctx)
			}
		} else {
			ctx.Notifications().New(app.Notification{
				Title: "Registration error",
				Body:  "No credential returned.",
			})
		}
		return nil
	})).Call("catch", app.FuncOf(func(this app.Value, p []app.Value) interface{} {
		if len(p) > 0 {
			// err := p[0]
			// Attempt to read the error message
			// var errorMessage string
			// if err.Get("message").String() != "" {
			// 	errorMessage = err.Get("message").String() // Standard way to get the message
			// } else if err.Get("error").String() != "" {
			// 	errorMessage = err.Get("error").String() // Some errors might have an 'error' property
			// } else {
			// 	errorMessage = "Unknown error occurred."
			// }

			// Notify user through UI instead of terminating application
			ctx.Notifications().New(app.Notification{
				Title: "Registration error",
				Body:  "Credential creation failed: " + a.webAuthn.Config.RPID,
			})
		} else {
			ctx.Notifications().New(app.Notification{
				Title: "Registration error",
				Body:  "Unknown error occurred.",
			})
		}

		return nil
	}))
}

func (a *auth) beginLogin(ctx app.Context) {
	// Generate a random challenge
	// Create a new random source seeded with the current time
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	challengeByteArray := make([]byte, 32)
	r.Read(challengeByteArray)
	challenge := app.Window().Get("Uint8Array").New(len(challengeByteArray)) // Generate a random challenge
	// Fill the Uint8Array with the byte values
	for i, b := range challengeByteArray {
		challenge.SetIndex(i, b)
	}

	// Create the allowCredentials array
	allowCredentials := app.Window().Get("Array").New(0) // Start with an empty array

	// Create the credential descriptor object
	credDescriptor := app.Window().Get("Object").New()
	credDescriptor.Set("type", "public-key")

	// Convert credentialID to Uint8Array (if it isn't already)
	// Assuming credentialID is a *string* representation of the ID, if not then this conversion is not needed
	encoder := app.Window().Get("TextEncoder").New()
	credentialIDUint8Array := encoder.Call("encode", a.credentialID)
	credDescriptor.Set("id", credentialIDUint8Array)

	// Add the credential descriptor to the allowCredentials array
	allowCredentials.Call("push", credDescriptor)
	obj := app.Window().Get("Object").New()
	obj.Set("challenge", challenge)
	obj.Set("rpId", a.webAuthn.Config.RPID)
	obj.Set("userVerification", "required")
	obj.Set("allowCredentials:", allowCredentials)
	obj.Set("publicKey", obj)

	// Access the navigator object
	promise := app.Window().Get("navigator").Get("credentials").Call("get", obj)

	// Step 3: Handle the promise response
	promise.Call("then", app.FuncOf(func(this app.Value, args []app.Value) interface{} {
		if len(args) > 0 {
			ctx.Notifications().New(app.Notification{
				Title: "Success",
				Body:  "Login successful!",
			})
			ctx.SetState("loggedIn", true)
			// redirect to wallet
			ctx.Navigate("/wallet")
		} else {
			ctx.Notifications().New(app.Notification{
				Title: "Login error",
				Body:  "No credential returned.",
			})
			log.Fatal("No credential returned")
		}
		return nil
	})).Call("catch", app.FuncOf(func(this app.Value, p []app.Value) interface{} {
		if len(p) > 0 {
			// err := p[0]
			// // Attempt to read the error message
			// var errorMessage string
			// if err.Get("message").String() != "" {
			// 	errorMessage = err.Get("message").String() // Standard way to get the message
			// } else if err.Get("error").String() != "" {
			// 	errorMessage = err.Get("error").String() // Some errors might have an 'error' property
			// } else {
			// 	errorMessage = "Unknown error occurred."
			// }

			// Notify user through UI instead of terminating application
			ctx.Notifications().New(app.Notification{
				Title: "Login error",
				Body:  "Credential fetching failed: " + a.webAuthn.Config.RPID,
			})
		} else {
			ctx.Notifications().New(app.Notification{
				Title: "Login error",
				Body:  "Unknown error occurred.",
			})
		}

		return nil
	}))
}

// The Render method is where the component appearance is defined. Here, a
// webauthn is displayed.
func (a *auth) Render() app.UI {
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
				app.Div().Class("card card-auth").Body(
					app.Div().Class("upper-row").Body(
						app.Div().Class("card-item").Body(
							app.Span().Class("span-header").Text("Face ID"),
						),
					),
					app.Div().Class("lower-row").Body(
						app.Div().Class("card-item").Body(
							app.Div().Class("container").Body(
								app.Video().ID("video").Width(225).Height(225).AutoPlay(true).Muted(true),
								app.Canvas().ID("canvas").Width(225).Height(225),
							),
						),
					),
				),
				app.Div().Class("drawer drawer-auth").Body(
					app.Div().ID("auth-bar").Class("auth-bar").Body(
						app.Span().Class("auth-message").Text("Authenticating"),
						app.Span().Class("blinking").Text("..."),
					),
					app.Input().ID("check-btn").OnClick(a.doCheck).Hidden(true),
				),
			),
		),
	)
}
