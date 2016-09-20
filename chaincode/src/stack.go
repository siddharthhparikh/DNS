/*
Copyright IBM Corp 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Contributors:
 Justin E. Ervin - Initial implementation
*/

package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	//"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	//"strconv"
	"strings"
	"time"
	"math/rand"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// SimpleChaincode example simple Chaincode implementation
type DNSChaincode struct {
}

type DomainName struct {
	userEmail 	string `json:"userEmail"`
	address 	string `json:"ipAddress"`
}

type IPAddress struct {
	userEmail 	string `json:"userEmail"`
	domainName 	string `json:"domainName"`
}

type account struct {
	email 				string 	`json:"email"`
	registrationDate 	string 	`json:"reg_date"`
	pubKey 				string 	`json:"pub_key"`
}
// ============================================================================================================================
// Main
// ============================================================================================================================
func main() {
	err := shim.Start(new(DNSChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// Init resets all the things
func (t *DNSChaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	if len(args) != 0 {
		return nil, errors.New("Incorrect number of arguments. Expecting 0")
	}

	if function == "init" {
		var err error

		fmt.Println("Creating the DNS look up table...")
		err = stub.CreateTable("NameToIP", []*shim.ColumnDefinition{
			{"domainName", shim.ColumnDefinition_STRING, true},
			{"ipAddress", shim.ColumnDefinition_STRING, false},
			{"userEmail", shim.ColumnDefinition_STRING, false},
			{"DateRegistered", shim.ColumnDefinition_STRING, false},
			{"Duration", shim.ColumnDefinition_STRING, false},
		})
		if err != nil {
			fmt.Println("Error creating table: ", err)
		}

		fmt.Println("Creating the IP address to Name table...")
		err = stub.CreateTable("IPToName", []*shim.ColumnDefinition{
			{"ipAddress", shim.ColumnDefinition_STRING, true},
			{"domainName", shim.ColumnDefinition_STRING, false},
			{"userEmail", shim.ColumnDefinition_STRING, false},
			{"DateRegistered", shim.ColumnDefinition_STRING, false},
			{"Duration", shim.ColumnDefinition_STRING, false},
		})
		if err != nil {
			fmt.Println("Error creating table: ", err)
		}

		err = stub.CreateTable("TransferRequests", []*shim.ColumnDefinition{
			{"RequestID", shim.ColumnDefinition_STRING, true},
			{"Owner", shim.ColumnDefinition_STRING, false},
			{"Buyer", shim.ColumnDefinition_STRING, false},
			{"BidValue", shim.ColumnDefinition_STRING, false},
			{"Status", shim.ColumnDefinition_STRING, false},
			{"DateRequested", shim.ColumnDefinition_STRING, false},
			{"DateDecision", shim.ColumnDefinition_STRING, false},
			{"DomainName", shim.ColumnDefinition_STRING, false},
		})
		if err != nil {
			fmt.Println("Error creating table: ", err)
		}

		err = stub.CreateTable("RegisteredUsers", []*shim.ColumnDefinition{
			{"userEmail", shim.ColumnDefinition_STRING, true},
			{"PubKey", shim.ColumnDefinition_STRING, false},
			{"RegistrationDate", shim.ColumnDefinition_STRING, false},
			{"DomainOwned", shim.ColumnDefinition_STRING, false},
			{"RequestedBids", shim.ColumnDefinition_STRING, false},
			{"OwnedBids", shim.ColumnDefinition_STRING, false},
		})
		if err != nil {
			fmt.Println("Error creating table: ", err)
		}
	}
	return nil, nil
}

func (t *DNSChaincode) getUserPubKey(stub *shim.ChaincodeStub, args []string) (string) {
	userEmail := args[0]
	accountRow, accountErr := stub.GetRow("RegisteredUsers", []shim.Column{{Value: &shim.Column_String_{String_: userEmail}}})
	if accountErr != nil {
		return ""	
	} else if len(accountRow.Columns) == 0 {
		return ""	
	} else {
		return accountRow.Columns[1].GetString_()
	}
}

type Unsigner interface {
	Unsign(data []byte, sig []byte) error
}

func (r *rsaPublicKey) Unsign(message []byte, sig []byte) error {
	h := sha256.New()
	h.Write(message)
	d := h.Sum(nil)
	return rsa.VerifyPKCS1v15(r.PublicKey, crypto.SHA256, d, sig)
}

func (t *DNSChaincode) newUnsignerFromKey(k interface{}) (Unsigner, error) {
	var sshKey Unsigner
	switch t := k.(type) {
	case *rsa.PublicKey:
		sshKey = &rsaPublicKey{t}
	default:
		return nil, fmt.Errorf("ssh: unsupported key type %T", k)
	}
	return sshKey, nil
}

type rsaPublicKey struct {
	*rsa.PublicKey
}

func (t *DNSChaincode) parsePublicKey(pemBytes []byte) (Unsigner, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("ssh: no key found")
	}

	var rawkey interface{}
	switch block.Type {
	case "PUBLIC KEY":
		rsa, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}

		rawkey = rsa
	default:
		return nil, fmt.Errorf("ssh: unsupported key type %q", block.Type)
	}

	return t.newUnsignerFromKey(rawkey)
}

func (t *DNSChaincode) checkUserPrivKey(stub *shim.ChaincodeStub, args []string) (bool) {
	//pubKey := t.getUserPubKey(stub,args)
	//signature := []byte(args[1])
	signByte, _ := hex.DecodeString(args[1])
	pubByte, _ := hex.DecodeString(t.getUserPubKey(stub,args))
	key, keyError := t.parsePublicKey(pubByte)
	if keyError == nil {
		return key.Unsign([]byte(args[0]), signByte) == nil
	}
	return false
}
// Invoke is our entry point to invoke a chaincode function
func (t *DNSChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	if t.checkUserPrivKey(stub,args) == false {
		return nil, errors.New("Signed by wrong private key. I can smell something fishy")
	}

	// Handle different functions
	if function == "init" {
		return t.Init(stub, "init", args)
	} else if function == "createAccount" {
		return t.createAccount(stub, args)
	} else if function == "registerDomain" {
		return t.registerDomain(stub, args)
	} else if function == "transferDomain" {
		return t.transferDomain(stub, args)
	} else if function == "placeBid" {
		return t.placeBid(stub, args)
	}

	fmt.Println("invoke did not find func: " + function)
	return nil, errors.New("Received unknown function invocation")
}

func (t *DNSChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {


	fmt.Println("query did not find func: " + function)
	return nil, errors.New("Received unknown function query")
}

func generateRandomNumber() string {
	//random number generator
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const (
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 73 / letterIdxBits   // # of letter indices fitting in 63 bits
	)
	n := 6
	var src = rand.NewSource(time.Now().UnixNano())
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return string(b)
}

func (t *DNSChaincode) getUniqueID(stub *shim.ChaincodeStub, i int) (string, error) {
	randomID := generateRandomNumber()
	requestRow, err := stub.GetRow("TransferRequests", []shim.Column{{Value: &shim.Column_String_{String_: randomID}}})
	if (err != nil) {
		return "", err
	} else {
		if len(requestRow.Columns) != 0 || i!=0 {
			return t.getUniqueID(stub, i-1)
		} else if i==0 {
			return "", errors.New("Cannot generate unique random ID for the transection after 10 attempts. Try again.")
		} else if len(requestRow.Columns) == 0 {
			return randomID, nil
		}
	}
	return "",errors.New("Returning without any error but there is an error. Code should not reach this part.")
}
func (t *DNSChaincode) placeBid(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	fromBid := args[0]
	toBid := args[2]
	domainName := args[3]
	amount := args[4]

	transectionID, err := t.getUniqueID(stub, 10) //try 10 times to generate a random ID
	if err!=nil {
		return nil, err
	}
	row, err := stub.GetRow("TransferRequests", []shim.Column{{Value: &shim.Column_String_{String_: transectionID}}})
	if err != nil || len(row.Columns) == 0 {
		rowAdded, rowErr := stub.InsertRow("TransferRequests", shim.Row{
			Columns: []*shim.Column{
				&shim.Column{Value: &shim.Column_String_{String_: transectionID}},
				&shim.Column{Value: &shim.Column_String_{String_: toBid}},
				&shim.Column{Value: &shim.Column_String_{String_: fromBid}},
				&shim.Column{Value: &shim.Column_String_{String_: amount}},
				&shim.Column{Value: &shim.Column_String_{String_: "open"}},
				&shim.Column{Value: &shim.Column_String_{String_: time.Now().Format("02 Jan 06 15:04 MST")}},
				&shim.Column{Value: &shim.Column_String_{String_: ""}},
				&shim.Column{Value: &shim.Column_String_{String_: domainName}},
			},
		})

		if rowErr != nil || !rowAdded {
			return nil, errors.New(fmt.Sprintf("Error creating row: %s", err))
		}
	} else {
		return nil, errors.New("Something went wrong. TransectionID already exist. Cannot edit existing ID. Please place bid again.")
	}

	//Update accounts of owner and potential buyer
	accountRow, accountErr := stub.GetRow("RegisteredUsers", []shim.Column{{Value: &shim.Column_String_{String_: toBid}}})	
	if accountErr != nil {
		return nil, accountErr	
	} else if len(accountRow.Columns) == 0 {
		return nil, errors.New("Account does not exists. Not sure how did you get this far but its time to go back and register.")	
	} else {
		_, err = stub.ReplaceRow("RegisteredUsers", shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: accountRow.Columns[0].GetString_()}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[1].GetString_()}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[2].GetString_()}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[3].GetString_()}},
				{Value: &shim.Column_String_{String_: strings.Join([]string{accountRow.Columns[4].GetString_(),transectionID},",")}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[5].GetString_()}},
			},
		})

		if err != nil {
			return nil, errors.New(fmt.Sprintf("Error updating row for the profile: %s", err))
		}
	}

	accountRow, accountErr = stub.GetRow("RegisteredUsers", []shim.Column{{Value: &shim.Column_String_{String_: fromBid}}})
	if accountErr != nil {
		return nil, accountErr	
	} else if len(accountRow.Columns) == 0 {
		return nil, errors.New("Account does not exists. Not sure how did you get this far but its time to go back and register.")	
	} else {
		_, err = stub.ReplaceRow("RegisteredUsers", shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: accountRow.Columns[0].GetString_()}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[1].GetString_()}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[2].GetString_()}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[3].GetString_()}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[4].GetString_()}},
				{Value: &shim.Column_String_{String_: strings.Join([]string{accountRow.Columns[5].GetString_(),transectionID},",")}},
			},
		})

		if err != nil {
			return nil, errors.New(fmt.Sprintf("Error updating row for the profile: %s", err))
		}
	}

	return nil, nil
}
func (t *DNSChaincode) createAccount(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

	//args[0] = emailID
	//args[2] = public key
	acc := account{email: args[0], registrationDate: time.Now().Format("02 Jan 06 15:04 MST"), pubKey: args[2]} 
	accountRow, err := stub.GetRow("RegisteredUsers", []shim.Column{{Value: &shim.Column_String_{String_: acc.email}}})
	if err != nil || len(accountRow.Columns) == 0 {
		rowAdded, rowErr := stub.InsertRow("RegisteredUsers", shim.Row{
			Columns: []*shim.Column{
				&shim.Column{Value: &shim.Column_String_{String_: acc.email}},
				&shim.Column{Value: &shim.Column_String_{String_: acc.pubKey}},
				&shim.Column{Value: &shim.Column_String_{String_: acc.registrationDate}},
				&shim.Column{Value: &shim.Column_String_{String_: ""}},
				&shim.Column{Value: &shim.Column_String_{String_: ""}},
				&shim.Column{Value: &shim.Column_String_{String_: ""}},
			},
		})

		if rowErr != nil || !rowAdded {
			return nil, errors.New(fmt.Sprintf("Error creating row: %s", err))
		}
	} else {
		return nil, errors.New("Account already exists. Please login.")
	}

	return nil, nil
}
func (t *DNSChaincode) registerDomain(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	
	userEmail := args[0]
	domainName := args[2]
	ipAddress := args[3]
	registrationDate := time.Now().Format("02 Jan 06 15:04 MST")
	duration := args[4]

	//Update Name to IP lookup table as well as IP to Name. 
	domainRow, err := stub.GetRow("NameToIP", []shim.Column{{Value: &shim.Column_String_{String_: domainName}}})
	if err != nil || len(domainRow.Columns) == 0 {
		rowAdded, rowErr := stub.InsertRow("NameToIP", shim.Row{
			Columns: []*shim.Column{
				&shim.Column{Value: &shim.Column_String_{String_: domainName}},
				&shim.Column{Value: &shim.Column_String_{String_: ipAddress}},
				&shim.Column{Value: &shim.Column_String_{String_: userEmail}},
				&shim.Column{Value: &shim.Column_String_{String_: registrationDate}},
				&shim.Column{Value: &shim.Column_String_{String_: duration}},
			},
		})

		if rowErr != nil || !rowAdded {
			return nil, errors.New(fmt.Sprintf("Error creating row: %s", err))
		}
	} else {
		return nil, errors.New("Domain already exists. Please request a transfer.")
	}

	ipRow, ipErr := stub.GetRow("IPToName", []shim.Column{{Value: &shim.Column_String_{String_: ipAddress}}})
	if ipErr != nil || len(ipRow.Columns) == 0 {
		rowAdded, rowErr := stub.InsertRow("IPToName", shim.Row{
			Columns: []*shim.Column{
				{Value: &shim.Column_String_{String_: ipAddress}},
				{Value: &shim.Column_String_{String_: domainName}},
				{Value: &shim.Column_String_{String_: userEmail}},
				{Value: &shim.Column_String_{String_: registrationDate}},
				{Value: &shim.Column_String_{String_: duration}},
			},
		})

		if rowErr != nil || !rowAdded {
			return nil, errors.New(fmt.Sprintf("Error creating row: %s", err))
		}
	} else {
		return nil, errors.New("IP address is already assigned to another domain name. Please select a new IP address.")
	}

	accountRow, accountErr := stub.GetRow("RegisteredUsers", []shim.Column{{Value: &shim.Column_String_{String_: userEmail}}})
	
	if accountErr != nil {
		return nil, err	
	} else if len(accountRow.Columns) == 0 {
		return nil, errors.New("Account does not exists. Not sure how did you get this far but its time to go back and register.")	
	} else {
		_, err = stub.ReplaceRow("RegisteredUsers", shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: accountRow.Columns[0].GetString_()}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[1].GetString_()}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[2].GetString_()}},
				{Value: &shim.Column_String_{String_: strings.Join([]string{accountRow.Columns[3].GetString_(),domainName},",")}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[4].GetString_()}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[5].GetString_()}},
			},
		})

		if err != nil {
			return nil, errors.New(fmt.Sprintf("Error updating row for the profile: %s", err))
		}
	}

	return nil, nil
}
func (t *DNSChaincode) getRequestID(stub *shim.ChaincodeStub, args []string) (string, error) {
	
	oldOwner := args[0]
	rowChan, err := stub.GetRows("TransferRequests", []shim.Column{})
	if err!=nil {
		return "", err
	}
	for chanValue := range rowChan {
		if(chanValue.Columns[1].GetString_() == oldOwner) {
			return chanValue.Columns[0].GetString_(), nil
		}
	}
	return "", errors.New("Could not find request ID. Please check if transfer request is submitted or not")
}

func (t *DNSChaincode) transferDomain(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	
	oldOwner := args[0]
	domainName := args[2]
	newOwner := args[3]
	newIP := args[4]

	requestID, err := t.getRequestID(stub, args)
	if err!= nil {
		return nil, err
	}
	
	transferRow, transferErr := stub.GetRow("TransferRequests", []shim.Column{{Value: &shim.Column_String_{String_: requestID}}})
	if transferErr != nil || len(transferRow.Columns) == 0 {	
		_, err = stub.ReplaceRow("TransferRequests", shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: transferRow.Columns[0].GetString_()}},
				{Value: &shim.Column_String_{String_: transferRow.Columns[1].GetString_()}},
				{Value: &shim.Column_String_{String_: transferRow.Columns[2].GetString_()}},
				{Value: &shim.Column_String_{String_: transferRow.Columns[3].GetString_()}},
				{Value: &shim.Column_String_{String_: "transfer accepted"}},
				{Value: &shim.Column_String_{String_: transferRow.Columns[5].GetString_()}},
				{Value: &shim.Column_String_{String_: time.Now().Format("02 Jan 06 15:04 MST")}},
				{Value: &shim.Column_String_{String_: domainName}},
			},
		})
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Error updating row: %s", err))
		}
	} else {
		return nil, errors.New("Transfer ID does not exists. Please check transfer request.")	
	}

	accountRow, accountErr := stub.GetRow("RegisteredUsers", []shim.Column{{Value: &shim.Column_String_{String_: oldOwner}}})
	if accountErr != nil || len(accountRow.Columns) == 0 {
		return nil, errors.New("Account does not exists. Not sure how did you get this far but its time to go back and register.")	
	} else {
		arr := strings.Split(accountRow.Columns[3].GetString_(), ",")
		temp := ""
		for i:=0; i<len(arr); i++ {
			if(arr[i] != domainName) {
				strings.Join([]string{temp,arr[i]},",")
			}
		}

		arr = strings.Split(accountRow.Columns[4].GetString_(), ",")
		temp2 := ""
		for i:=0; i<len(arr); i++ {
			if(arr[i] != requestID) {
				strings.Join([]string{temp2,arr[i]},",")
			}
		}

		_, err = stub.ReplaceRow("RegisteredUsers", shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: accountRow.Columns[0].GetString_()}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[1].GetString_()}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[2].GetString_()}},
				{Value: &shim.Column_String_{String_: temp}},
				{Value: &shim.Column_String_{String_: temp2}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[5].GetString_()}},
			},
		})

		if err != nil {
			return nil, errors.New(fmt.Sprintf("Error updating row for the profile: %s", err))
		}
	}

	accountRow, accountErr = stub.GetRow("RegisteredUsers", []shim.Column{{Value: &shim.Column_String_{String_: newOwner}}})
	if accountErr != nil || len(accountRow.Columns) == 0 {
		return nil, errors.New("Account does not exists. Not sure how did you get this far but its time to go back and register.")	
	} else {
		arr := strings.Split(accountRow.Columns[5].GetString_(), ",")
		temp := ""
		for i:=0; i<len(arr); i++ {
			if(arr[i] != domainName) {
				strings.Join([]string{temp,arr[i]},",")
			}
		}

		_, err = stub.ReplaceRow("RegisteredUsers", shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: accountRow.Columns[0].GetString_()}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[1].GetString_()}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[2].GetString_()}},
				{Value: &shim.Column_String_{String_: strings.Join([]string{accountRow.Columns[3].GetString_(),domainName},",")}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[4].GetString_()}},
				{Value: &shim.Column_String_{String_: temp}},
			},
		})

		if err != nil {
			return nil, errors.New(fmt.Sprintf("Error updating row for the profile: %s", err))
		}
	}

	//Add new IP and domain to IP and domain Table

	var oldIP string
	nameRow, nameErr := stub.GetRow("NameToIP", []shim.Column{{Value: &shim.Column_String_{String_: domainName}}})
	if nameErr != nil || len(nameRow.Columns) == 0 {
		return nil, errors.New("Domain does not exists. Not sure how did you get this far but its time to go back and register.")	
	} else {
		oldIP = nameRow.Columns[1].GetString_()
		_, err = stub.ReplaceRow("NameToIP", shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: domainName}},
				{Value: &shim.Column_String_{String_: newIP}},
				{Value: &shim.Column_String_{String_: newOwner}},
				{Value: &shim.Column_String_{String_: time.Now().Format("02 Jan 06 15:04 MST")}},
				{Value: &shim.Column_String_{String_: nameRow.Columns[4].GetString_()}},
			},
		})
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Error updating row for the profile: %s", err))
		}
	}

	ipRow, ipErr := stub.GetRow("IPToName", []shim.Column{{Value: &shim.Column_String_{String_: oldIP}}})
	if ipErr != nil || len(ipRow.Columns) == 0 {
		return nil, errors.New("Something is not quite right. This should not have happened.")	
	} else {
		_, err = stub.ReplaceRow("IPToName", shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: newIP}},
				{Value: &shim.Column_String_{String_: ipRow.Columns[1].GetString_()}},
				{Value: &shim.Column_String_{String_: newOwner}},
				{Value: &shim.Column_String_{String_: time.Now().Format("02 Jan 06 15:04 MST")}},
				{Value: &shim.Column_String_{String_: ipRow.Columns[4].GetString_()}},
			},
		})

		if err != nil {
			return nil, errors.New(fmt.Sprintf("Error updating row for the profile: %s", err))
		}
	}

	return nil, nil
}
