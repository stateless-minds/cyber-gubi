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

func (a *auth) deleteIncome() {
	err := a.sh.OrbitDocsDelete(dbIncome, "all")
	if err != nil {
		log.Fatal(err)
	}
}

func (a *auth) deleteTransactions() {
	err := a.sh.OrbitDocsDelete(dbTransaction, "all")
	if err != nil {
		log.Fatal(err)
	}
}

func (a *auth) deleteInflation() {
	err := a.sh.OrbitDocsDelete(dbInflation, "all")
	if err != nil {
		log.Fatal(err)
	}
}

func (a *auth) getInflation() {
	_, err := a.sh.OrbitDocsQuery(dbInflation, "all", "")
	if err != nil {
		log.Fatal(err)
	}
}

func (a *auth) deletePlans() {
	err := a.sh.OrbitDocsDelete(dbPlan, "all")
	if err != nil {
		log.Fatal(err)
	}
}

func (a *auth) deleteSubscriptions() {
	err := a.sh.OrbitDocsDelete(dbSubscription, "all")
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
			log.Println("no income set")
		} else {
			err = json.Unmarshal([]byte(i), &income) // Unmarshal the byte slice directly
			if err != nil {
				log.Fatal(err)
			}
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

			a.createUser(ctx)
		} else {
			ctx.Notifications().New(app.Notification{
				Title: "Registration error",
				Body:  "No credential returned.",
			})
		}
		return nil
	})).Call("catch", app.FuncOf(func(this app.Value, p []app.Value) interface{} {
		if len(p) > 0 {
			err := p[0]
			// Attempt to read the error message
			var errorMessage string
			if err.Get("message").String() != "" {
				errorMessage = err.Get("message").String() // Standard way to get the message
			} else if err.Get("error").String() != "" {
				errorMessage = err.Get("error").String() // Some errors might have an 'error' property
			} else {
				errorMessage = "Unknown error occurred."
			}

			// Notify user through UI instead of terminating application
			ctx.Notifications().New(app.Notification{
				Title: "Registration error",
				Body:  "Credential creation failed: " + errorMessage,
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
			err := p[0]
			// Attempt to read the error message
			var errorMessage string
			if err.Get("message").String() != "" {
				errorMessage = err.Get("message").String() // Standard way to get the message
			} else if err.Get("error").String() != "" {
				errorMessage = err.Get("error").String() // Some errors might have an 'error' property
			} else {
				errorMessage = "Unknown error occurred."
			}

			// Notify user through UI instead of terminating application
			ctx.Notifications().New(app.Notification{
				Title: "Login error",
				Body:  "Credential fetching failed: " + errorMessage,
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
