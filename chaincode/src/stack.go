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
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

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
			{"RequestID", shim.ColumnDefinition_STRING, true}
			{"Owner", shim.ColumnDefinition_STRING, false},
			{"Buyer", shim.ColumnDefinition_STRING, false},
			{"Amount", shim.ColumnDefinition_STRING, false},
			{"Status", shim.ColumnDefinition_STRING, false},
			{"DateRequested", shim.ColumnDefinition_STRING, false},
			{"DateDecision", shim.ColumnDefinition_STRING, false},
			{"BidValue", shim.ColumnDefinition_STRING, false},
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

	return nil, nil
}

// Invoke is our entry point to invoke a chaincode function
func (t *DNSChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

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
func (t *DNSChaincode) createAccount(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

	//args[0] = emailID
	//args[1] = public key
	acc := account{email: args[0], registrationDate: time.Now().Format("02 Jan 06 15:04 MST"), pubkey: args[1]} 
	accountRow, err := stub.GetRow("RegisteredUsers", []shim.Column{{Value: &shim.Column_String_{String_: acc.email}}})
	if err != nil || len(accountRow.Columns) == 0 {
		rowAdded, rowErr := stub.InsertRow("RegisteredUsers", shim.Row{
			Columns: []*shim.Column{
				&shim.Column{Value: &shim.Column_String_{String_: acc.email}},
				&shim.Column{Value: &shim.Column_String_{String_: acc.pubkey}},
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
}
func (t *DNSChaincode) registerDomain(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	
	domainName := args[0]
	ipAddress := args[1]
	userEmail := args[2]
	registrationDate := time.Now().Format("02 Jan 06 15:04 MST")
	duration := args[3]

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
				{Value: &shim.Column_String_{String_: strings.Join([]string{accountRow.Columns[3].GetString_(),',',domainName)}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[4].GetString_()}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[5].GetString_()}},
			},
		})

		if err != nil {
			return errors.New(fmt.Sprintf("Error updating row for the profile: %s", err))
		}
	}
}
func (t *DNSChaincode) getRequestID(stub *shim.ChaincodeStub, args []string) (string, error) {
	
	oldOwner := args[1]
	rowChan, err := stub.GetRows("TransferRequests", []shim.Column{})
	for chanValue := range rowChan {
		if(chanValue.Columns[1] == oldOwner) {
			return chanValue.Columns[0], nil
		}
	}
	return "", errors.New("Could not find request ID. Please check if transfer request is submitted or not")
}

func (t *DNSChaincode) transferDomain(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	domainName := args[0]
	oldOwner := args[1]
	newOwner := args[2]
	newIP := args[3]

	requestID, err := t.getRequestID(stub, args)
	if err!= nil {
		return nil, err
	}
	
	transferRow, transferErr := stub.GetRow("TransferRequests", []shim.Column{{Value: &shim.Column_String_{String_: requestID}}})
	if transferErr != nil || len(transferRow.Columns) == 0 {	
		_, err = stub.ReplaceRow("TransferRequests", shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: accountRow.Columns[0].GetString_()}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[1].GetString_()}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[2].GetString_()}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[3].GetString_()}},
				{Value: &shim.Column_String_{String_: "transfer accepted"}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[5].GetString_()}},
				{Value: &shim.Column_String_{String_: time.Now().Format("02 Jan 06 15:04 MST")}},
				{Value: &shim.Column_String_{String_: accountRow.Columns[7].GetString_()}},
			}
		})
		if err != nil {
			return errors.New(fmt.Sprintf("Error updating row: %s", err))
		}
	} else {
		return nil, errors.New("Transfer ID does not exists. Please check transfer request.")	
	}

	accountRow, accountErr := stub.GetRow("RegisteredUsers", []shim.Column{{Value: &shim.Column_String_{String_: oldOwner}}})
	if accountErr != nil || len(accountRow.Columns) == 0 {
		return nil, errors.New("Account does not exists. Not sure how did you get this far but its time to go back and register.")	
	} else {
			/*
			{"userEmail", shim.ColumnDefinition_STRING, true},
			{"PubKey", shim.ColumnDefinition_STRING, false},
			{"RegistrationDate", shim.ColumnDefinition_STRING, false},
			{"DomainOwned", shim.ColumnDefinition_STRING, false},
			{"RequestedBids", shim.ColumnDefinition_STRING, false},
			{"OwnedBids", shim.ColumnDefinition_STRING, false},
			*/
		arr := strings.Split(accountRow.Columns[3].GetString_(), ",")
		temp := ""
		for i:=0; i<len(arr); i++ {
			if(arr[i] != domainName) {
				strings.Join([]string{temp,',',arr[i]})
			}
		}

		arr = strings.Split(accountRow.Columns[4].GetString_(), ",")
		temp2 := ""
		for i=0; i<len(arr); i++ {
			if(arr[i] != requestID) {
				strings.Join([]string{temp2,',',arr[i]})
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
			return errors.New(fmt.Sprintf("Error updating row for the profile: %s", err))
		}
	}
}

func (t *DNSChaincode) profileInit(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args) < 4 {
		return nil, errors.New("Incorrect number of arguments. Expecting 4")
	}

	// args[0] - UserID
	// args[1] - Public Key
	// args[2] - Json Data
	// args[3] - Credits

	var idRow, profileRow shim.Row
	var rowAdded bool
	var err error

	idRow, err = stub.GetRow(REGISTERED_PROFILE_ID_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: strings.TrimSpace(args[1])}}})
	if err != nil || len(idRow.Columns) == 0 {
		rowAdded, err = stub.InsertRow(REGISTERED_PROFILE_ID_TABLE, shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: strings.TrimSpace(strings.Replace(args[1], " ", "", -1))}},
				{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}},
			},
		})

		if err != nil || !rowAdded {
			return nil, errors.New(fmt.Sprintf("Error creating row: %s", err))
		}
	} else {
		return nil, errors.New("Public key already exists in the profile table")
	}

	profileRow, err = stub.GetRow(REGISTERED_PROFILE_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}}})
	if err != nil || len(profileRow.Columns) == 0 {
		rowAdded, err = stub.InsertRow(REGISTERED_PROFILE_TABLE, shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}},
				{Value: &shim.Column_String_{String_: strings.TrimSpace(strings.Replace(args[1], " ", "", -1))}},
				{Value: &shim.Column_Bytes{Bytes: []byte(strings.TrimSpace(args[2]))}},
				{Value: &shim.Column_String_{String_: ""}},
				{Value: &shim.Column_String_{String_: strings.TrimSpace(args[3])}},
			},
		})

		if err != nil || !rowAdded {
			return nil, errors.New(fmt.Sprintf("Error creating row: %s", err))
		}

		err = t.updateStats(stub, CURRENT_PROFILE_STATS_NAME, 1)
		if err != nil {
			return nil, fmt.Errorf("Error updating stats: %s", err)
		}
	} else {
		return nil, errors.New("User Id already exists in the profile table")
	}

	return nil, nil
}

func (t *DNSChaincode) updateStats(stub *shim.ChaincodeStub, statsName string, value int) error {
	var state []byte
	var converted int
	var err error

	state, err = stub.GetState(statsName)
	if err != nil {
		return fmt.Errorf("State does not exists: %s", err)
	}

	converted, err = strconv.Atoi(string(state))
	if err != nil {
		return fmt.Errorf("Failed to convert: %s", err)
	}

	err = stub.PutState(statsName, []byte(strconv.Itoa(converted+value)))
	if err != nil {
		return fmt.Errorf("Error putting state: %s", err)
	}

	return nil
}

func (t *DNSChaincode) profileUpdateInfo(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args) < 3 {
		return nil, errors.New("Incorrect number of arguments. Expecting 3")
	}

	// args[0] - Owner Public Key
	// args[1] - Owner Signature
	// args[2] - Json Data

	if !t.checkAssetOwnership(stub, "", strings.TrimSpace(args[1]), strings.TrimSpace(args[0]), args, true) {
		return nil, errors.New("Invalid ownership")
	}

	var err error

	err = t.profileUpdateJson(stub, strings.TrimSpace(args[0]), strings.TrimSpace(args[2]))
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (t *DNSChaincode) profileUpdateJson(stub *shim.ChaincodeStub, userKey string, jsonData string) error {
	var profileIdRow, profileRow shim.Row
	var err error

	profileIdRow, err = stub.GetRow(REGISTERED_PROFILE_ID_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: strings.TrimSpace(strings.Replace(userKey, " ", "", -1))}}})
	if err != nil || len(profileIdRow.Columns) == 0 {
		return fmt.Errorf("Failed to get state: %s", err)
	}

	profileRow, err = stub.GetRow(REGISTERED_PROFILE_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: profileIdRow.Columns[PROFILE_ID_USER_ID_COLUMN].GetString_()}}})
	if err == nil && len(profileRow.Columns) != 0 {
		_, err = stub.ReplaceRow(REGISTERED_PROFILE_TABLE, shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: profileRow.Columns[PROFILE_USER_ID_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: profileRow.Columns[PROFILE_PUBLIC_KEY_COLUMN].GetString_()}},
				{Value: &shim.Column_Bytes{Bytes: []byte(jsonData)}},
				{Value: &shim.Column_String_{String_: profileRow.Columns[PROFILE_OWNED_ASSETS_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: profileRow.Columns[PROFILE_CREDITS_COLUMN].GetString_()}},
			},
		})

		if err != nil {
			return errors.New(fmt.Sprintf("Error updating row for the profile: %s", err))
		}
	}

	return nil
}

func (t *DNSChaincode) profileUpdateCredits(stub *shim.ChaincodeStub, userKey string, creditAmount float64) error {
	var profileIdRow, profileRow shim.Row
	var convertedTotalCredits float64
	var err error

	profileIdRow, err = stub.GetRow(REGISTERED_PROFILE_ID_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: strings.TrimSpace(strings.Replace(userKey, " ", "", -1))}}})
	if err != nil || len(profileIdRow.Columns) == 0 {
		return fmt.Errorf("Failed to get state: %s", err)
	}

	profileRow, err = stub.GetRow(REGISTERED_PROFILE_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: profileIdRow.Columns[PROFILE_ID_USER_ID_COLUMN].GetString_()}}})
	if err == nil && len(profileRow.Columns) != 0 {
		convertedTotalCredits, err = strconv.ParseFloat(profileRow.Columns[PROFILE_CREDITS_COLUMN].GetString_(), 64)
		if err != nil {
			return fmt.Errorf("Failed to convert: %s", err)
		}

		_, err = stub.ReplaceRow(REGISTERED_PROFILE_TABLE, shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: profileRow.Columns[PROFILE_USER_ID_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: profileRow.Columns[PROFILE_PUBLIC_KEY_COLUMN].GetString_()}},
				{Value: &shim.Column_Bytes{Bytes: profileRow.Columns[PROFILE_JSON_DATA_COLUMN].GetBytes()}},
				{Value: &shim.Column_String_{String_: profileRow.Columns[PROFILE_OWNED_ASSETS_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: strconv.FormatFloat(convertedTotalCredits-creditAmount, 'f', 2, 64)}},
			},
		})

		if err != nil {
			return errors.New(fmt.Sprintf("Error updating row for the profile: %s", err))
		}
	}

	return nil
}

func (t *DNSChaincode) profileCheckCredits(stub *shim.ChaincodeStub, userKey string, creditAmount float64) error {
	var profileIdRow, profileRow shim.Row
	var err error

	profileIdRow, err = stub.GetRow(REGISTERED_PROFILE_ID_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: strings.TrimSpace(strings.Replace(userKey, " ", "", -1))}}})
	if err != nil || len(profileIdRow.Columns) == 0 {
		return fmt.Errorf("Failed to get state: %s", err)
	}

	profileRow, err = stub.GetRow(REGISTERED_PROFILE_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: profileIdRow.Columns[PROFILE_ID_USER_ID_COLUMN].GetString_()}}})
	if err == nil && len(profileRow.Columns) != 0 {
		var convertedTotalCredits float64

		convertedTotalCredits, err = strconv.ParseFloat(profileRow.Columns[PROFILE_CREDITS_COLUMN].GetString_(), 64)
		if err != nil {
			return fmt.Errorf("Failed to convert: %s", err)
		}

		if convertedTotalCredits >= creditAmount {
			return nil
		}
	}

	return errors.New("Insufficient funds")
}

func (t *DNSChaincode) profileUpdateAssets(stub *shim.ChaincodeStub, userKey string, assetName string, assetAction int) error {
	var profileIdRow, profileRow shim.Row
	var err error

	profileIdRow, err = stub.GetRow(REGISTERED_PROFILE_ID_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: strings.TrimSpace(strings.Replace(userKey, " ", "", -1))}}})
	if err != nil || len(profileIdRow.Columns) == 0 {
		return fmt.Errorf("Failed to get state: %s", err)
	}

	profileRow, err = stub.GetRow(REGISTERED_PROFILE_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: profileIdRow.Columns[PROFILE_ID_USER_ID_COLUMN].GetString_()}}})
	if err == nil && len(profileRow.Columns) != 0 {
		var assetList []string = strings.Split(t.readStringSafe(profileRow.Columns[PROFILE_OWNED_ASSETS_COLUMN]), ",")

		if assetAction == PROFILE_ASSETS_ADD {
			assetList = append(assetList, strings.TrimSpace(assetName))
		} else if assetAction == PROFILE_ASSETS_DELETE {
			var foundIndex int = -1

			for index := 0; index < len(assetList); index++ {
				if strings.ToLower(strings.TrimSpace(assetList[index])) == strings.ToLower(strings.TrimSpace(strings.TrimSpace(assetName))) {
					foundIndex = index
					break
				}
			}

			if foundIndex >= 0 {
				assetList = append(assetList[:foundIndex], assetList[foundIndex+1:]...)
			}
		}

		_, err = stub.ReplaceRow(REGISTERED_PROFILE_TABLE, shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: profileRow.Columns[PROFILE_USER_ID_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: profileRow.Columns[PROFILE_PUBLIC_KEY_COLUMN].GetString_()}},
				{Value: &shim.Column_Bytes{Bytes: profileRow.Columns[PROFILE_JSON_DATA_COLUMN].GetBytes()}},
				{Value: &shim.Column_String_{String_: strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.Join(assetList, ","), ","), ","))}},
				{Value: &shim.Column_String_{String_: profileRow.Columns[PROFILE_CREDITS_COLUMN].GetString_()}},
			},
		})

		if err != nil {
			return errors.New(fmt.Sprintf("Error updating row for the asset: %s", err))
		}
	}

	return nil
}

func (t *DNSChaincode) assetInit(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args) < 9 {
		return nil, errors.New("Incorrect number of arguments. Expecting 9")
	}

	// args[0] - Asset
	// args[1] - Signature
	// args[2] - Public Key
	// args[3] - Description
	// args[4] - Length
	// args[5] - Value
	// args[6] - Amount
	// args[7] - Use length for cost
	// args[8] - Rate for the cost of the length

	var convertedValue, convertedAmount, convertedRate float64
	var convertedLength int
	var rowAdded bool
	var err error

	if !t.checkAssetOwnership(stub, args[0], args[1], args[2], args, true) {
		return nil, errors.New("Invalid ownership")
	}

	convertedLength, err = strconv.Atoi(args[4])
	if err != nil {
		return nil, fmt.Errorf("Failed to convert: %s", err)
	}

	if convertedLength <= 0 {
		return nil, errors.New("The length must be greater than zero")
	}

	convertedValue, err = strconv.ParseFloat(args[5], 64)
	if err != nil {
		return nil, fmt.Errorf("Failed to convert: %s", err)
	}

	convertedAmount, err = strconv.ParseFloat(args[6], 64)
	if err != nil {
		return nil, fmt.Errorf("Failed to convert: %s", err)
	}

	var assetCost float64
	var checkForLength bool

	checkForLength, err = strconv.ParseBool(args[7])
	if err == nil {
		convertedRate, err = strconv.ParseFloat(args[8], 64)
		if err != nil {
			return nil, fmt.Errorf("Failed to convert: %s", err)
		}

		if convertedRate <= 0 {
			convertedRate = 1.0
		}

		if checkForLength {
			assetCost = convertedAmount * convertedValue * float64(convertedLength) * convertedRate
		} else {
			assetCost = convertedAmount * convertedValue
		}
	} else {
		return nil, err
	}

	err = t.profileCheckCredits(stub, strings.TrimSpace(strings.Replace(args[2], " ", "", -1)), assetCost)
	if err != nil {
		return nil, err
	}

	rowAdded, err = stub.InsertRow(REGISTERED_ASSETS_TABLE, shim.Row{
		[]*shim.Column{
			{Value: &shim.Column_String_{String_: strings.TrimSpace(strings.Replace(args[0], " ", "", -1))}},
			{Value: &shim.Column_String_{String_: strings.TrimSpace(args[3])}},
			{Value: &shim.Column_String_{String_: time.Now().Format(time.RFC822Z)}},
			{Value: &shim.Column_String_{String_: time.Now().Add(time.Hour * 24 * time.Duration(convertedLength)).Format(time.RFC822Z)}},
			{Value: &shim.Column_String_{String_: strings.TrimSpace(args[4])}},
			{Value: &shim.Column_String_{String_: time.Now().Format(time.RFC822Z)}},
			{Value: &shim.Column_String_{String_: strings.TrimSpace(args[6])}},
			{Value: &shim.Column_String_{String_: strings.TrimSpace(args[5])}},
			{Value: &shim.Column_Bool{Bool: false}},
			{Value: &shim.Column_String_{String_: strings.TrimSpace(strings.Replace(args[2], " ", "", -1))}},
			{Value: &shim.Column_String_{String_: strings.TrimSpace(args[1])}},
		},
	})

	if err != nil || !rowAdded {
		return nil, errors.New(fmt.Sprintf("Error creating row: %s", err))
	}

	rowAdded, err = stub.InsertRow(REGISTERED_ASSETS_ID_TABLE, shim.Row{
		[]*shim.Column{
			{Value: &shim.Column_String_{String_: strings.TrimSpace(strings.Replace(args[2], " ", "", -1))}},
			{Value: &shim.Column_String_{String_: strings.TrimSpace(strings.Replace(args[0], " ", "", -1))}},
			{Value: &shim.Column_String_{String_: strings.TrimSpace(args[3])}},
			{Value: &shim.Column_String_{String_: time.Now().Format(time.RFC822Z)}},
			{Value: &shim.Column_String_{String_: time.Now().Add(time.Hour * 24 * time.Duration(convertedLength)).Format(time.RFC822Z)}},
			{Value: &shim.Column_String_{String_: strings.TrimSpace(args[4])}},
			{Value: &shim.Column_String_{String_: time.Now().Format(time.RFC822Z)}},
			{Value: &shim.Column_String_{String_: strings.TrimSpace(args[6])}},
			{Value: &shim.Column_String_{String_: strings.TrimSpace(args[5])}},
			{Value: &shim.Column_Bool{Bool: false}},
			{Value: &shim.Column_String_{String_: strings.TrimSpace(args[1])}},
		},
	})

	if err != nil || !rowAdded {
		return nil, errors.New(fmt.Sprintf("Error creating row: %s", err))
	}

	err = t.profileUpdateAssets(stub, args[2], args[0], PROFILE_ASSETS_ADD)
	if err != nil {
		return nil, fmt.Errorf("Error updating assets: %s", err)
	}

	err = t.profileUpdateCredits(stub, strings.TrimSpace(args[2]), assetCost)
	if err != nil {
		return nil, fmt.Errorf("Error updating credits: %s", err)
	}

	err = t.updateStats(stub, CURRENT_ASSETS_STATS_NAME, 1)
	if err != nil {
		return nil, fmt.Errorf("Error updating stats: %s", err)
	}

	err = t.updateStats(stub, TOTAL_ASSETS_STATS_NAME, 1)
	if err != nil {
		return nil, fmt.Errorf("Error updating stats: %s", err)
	}

	return nil, nil
}

const (
	ASSET_TRANSFER_WAIT_FOR_OWNER     = "waitforowner"
	ASSET_TRANSFER_WAIT_FOR_NEW_OWNER = "waitfornewowner"
)

func (t *DNSChaincode) assetTransferInit(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args) < 6 {
		return nil, errors.New("Incorrect number of arguments. Expecting 6")
	}

	// args[0] - Asset
	// args[1] - OwnerSignature
	// args[2] - NewOwnerUsername
	// args[3] - Trade Value
	// args[4] - Trade Quantity
	// args[5] - New Asset Name

	if !t.checkAssetOwnership(stub, args[0], args[1], "", args, false) {
		return nil, errors.New("Invalid ownership")
	}

	var rowA, rowB, assetRow, tradeRow shim.Row
	var newOwnerPubKey string
	var err error
	var rowAdded bool

	assetRow, err = stub.GetRow(REGISTERED_ASSETS_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: args[0]}}})
	if err != nil || len(assetRow.Columns) == 0 {
		return nil, errors.New("Invalid asset name")
	}

	if !assetRow.Columns[ASSET_LOCKED_COLUMN].GetBool() {
		var pubkey string = assetRow.Columns[ASSET_PUBLIC_KEY_COLUMN].GetString_()
		var value string = strings.TrimSpace(args[3])
		var amount string = strings.TrimSpace(args[4])

		rowA, err = stub.GetRow(REGISTERED_PROFILE_ID_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: strings.TrimSpace(pubkey)}}})
		if err != nil || len(rowA.Columns) == 0 {
			return nil, errors.New("Invalid party A")
		}

		rowB, err = stub.GetRow(REGISTERED_PROFILE_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: strings.TrimSpace(args[2])}}})
		if err != nil || len(rowB.Columns) == 0 {
			return nil, errors.New("Invalid party B")
		}

		newOwnerPubKey = rowB.Columns[PROFILE_PUBLIC_KEY_COLUMN].GetString_()

		tradeRow, err = stub.GetRow(OPEN_TRADE_TABLE, []shim.Column{
			{Value: &shim.Column_String_{String_: strings.TrimSpace(pubkey)}},
			{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}},
		})

		if err != nil || len(tradeRow.Columns) == 0 {
			rowAdded, err = stub.InsertRow(OPEN_TRADE_TABLE, shim.Row{
				[]*shim.Column{
					{Value: &shim.Column_String_{String_: strings.TrimSpace(rowA.Columns[PROFILE_ID_USER_ID_COLUMN].GetString_())}},
					{Value: &shim.Column_String_{String_: strings.TrimSpace(pubkey)}},
					{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}},
					{Value: &shim.Column_String_{String_: ASSET_TRANSFER_WAIT_FOR_NEW_OWNER}},
					{Value: &shim.Column_String_{String_: strings.TrimSpace(args[2])}},
					{Value: &shim.Column_String_{String_: strings.TrimSpace(strings.Replace(newOwnerPubKey, " ", "", -1))}},
					{Value: &shim.Column_String_{String_: strings.TrimSpace(rowA.Columns[PROFILE_ID_USER_ID_COLUMN].GetString_())}},
					{Value: &shim.Column_String_{String_: strings.TrimSpace(args[5])}},
					{Value: &shim.Column_String_{String_: strings.TrimSpace(amount)}},
					{Value: &shim.Column_String_{String_: strings.TrimSpace(value)}},
					{Value: &shim.Column_String_{String_: strings.TrimSpace(value)}},
					{Value: &shim.Column_String_{String_: assetRow.Columns[ASSET_TIME_SPAN_COLUMN].GetString_()}},
					{Value: &shim.Column_String_{String_: strings.TrimSpace(args[1])}},
				},
			})

			if err != nil || !rowAdded {
				return nil, errors.New(fmt.Sprintf("Error creating row: %s", err))
			}

			rowAdded, err = stub.InsertRow(OPEN_TRADE_TABLE, shim.Row{
				[]*shim.Column{
					{Value: &shim.Column_String_{String_: strings.TrimSpace(args[2])}},
					{Value: &shim.Column_String_{String_: strings.TrimSpace(strings.Replace(newOwnerPubKey, " ", "", -1))}},
					{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}},
					{Value: &shim.Column_String_{String_: ASSET_TRANSFER_WAIT_FOR_NEW_OWNER}},
					{Value: &shim.Column_String_{String_: strings.TrimSpace(rowA.Columns[PROFILE_ID_USER_ID_COLUMN].GetString_())}},
					{Value: &shim.Column_String_{String_: strings.TrimSpace(pubkey)}},
					{Value: &shim.Column_String_{String_: strings.TrimSpace(rowA.Columns[PROFILE_ID_USER_ID_COLUMN].GetString_())}},
					{Value: &shim.Column_String_{String_: strings.TrimSpace(args[5])}},
					{Value: &shim.Column_String_{String_: strings.TrimSpace(amount)}},
					{Value: &shim.Column_String_{String_: strings.TrimSpace(value)}},
					{Value: &shim.Column_String_{String_: strings.TrimSpace(value)}},
					{Value: &shim.Column_String_{String_: assetRow.Columns[ASSET_TIME_SPAN_COLUMN].GetString_()}},
					{Value: &shim.Column_String_{String_: strings.TrimSpace(args[1])}},
				},
			})

			if err != nil || !rowAdded {
				return nil, errors.New(fmt.Sprintf("Error creating row: %s", err))
			}

			err = t.assetUpdateLock(stub, args[0], true)
			if err != nil {
				return nil, fmt.Errorf("Error updating asset: %s", err)
			}

			err = t.updateStats(stub, OPEN_ASSET_TRADE_STATS_NAME, 1)
			if err != nil {
				return nil, fmt.Errorf("Error updating stats: %s", err)
			}

			err = t.updateStats(stub, TOTAL_OPEN_ASSET_TRADE_STATS_NAME, 1)
			if err != nil {
				return nil, fmt.Errorf("Error updating stats: %s", err)
			}
		}
	} else {
		return nil, errors.New("Asset is locked")
	}

	return nil, nil
}

func (t *DNSChaincode) assetTransferAcceptChoice(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	var assetRow, tradeRow shim.Row
	var err error

	assetRow, err = stub.GetRow(REGISTERED_ASSETS_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: args[0]}}})
	if err != nil || len(assetRow.Columns) == 0 {
		return nil, errors.New("Invalid asset name")
	}

	var pubkey = assetRow.Columns[ASSET_PUBLIC_KEY_COLUMN].GetString_()

	tradeRow, err = stub.GetRow(OPEN_TRADE_TABLE, []shim.Column{
		{Value: &shim.Column_String_{String_: strings.TrimSpace(pubkey)}},
		{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}},
	})
	if err != nil && len(tradeRow.Columns) != 0 {
		return nil, fmt.Errorf("Row not found: %s", err)
	}

	if tradeRow.Columns[TRADE_STATUS_COLUMN].GetString_() == ASSET_TRANSFER_WAIT_FOR_OWNER {
		// args[0] - Asset
		// args[1] - Owner Signature

		return t.assetTransferAcceptByOwner(stub, args)
	} else if tradeRow.Columns[TRADE_STATUS_COLUMN].GetString_() == ASSET_TRANSFER_WAIT_FOR_NEW_OWNER {
		// args[0] - Asset
		// args[1] - New Owner Signature
		// args[2] - New Owner Public Key
		// args[3] - New Owner Asset Signature

		return t.assetTransferAcceptByNewOwner(stub, args)
	}

	return nil, nil
}

func (t *DNSChaincode) assetTransferDeclineChoice(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	var assetRow, tradeRow shim.Row
	var err error

	assetRow, err = stub.GetRow(REGISTERED_ASSETS_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: args[0]}}})
	if err != nil || len(assetRow.Columns) == 0 {
		return nil, errors.New("Invalid asset name")
	}

	var pubkey = assetRow.Columns[ASSET_PUBLIC_KEY_COLUMN].GetString_()

	tradeRow, err = stub.GetRow(OPEN_TRADE_TABLE, []shim.Column{
		{Value: &shim.Column_String_{String_: strings.TrimSpace(pubkey)}},
		{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}},
	})
	if err != nil && len(tradeRow.Columns) != 0 {
		return nil, fmt.Errorf("Row not found: %s", err)
	}

	if tradeRow.Columns[TRADE_STATUS_COLUMN].GetString_() == ASSET_TRANSFER_WAIT_FOR_OWNER {
		// args[0] - Asset
		// args[1] - Owner Signature

		return t.assetTransferDeclineByOwner(stub, args)
	} else if tradeRow.Columns[TRADE_STATUS_COLUMN].GetString_() == ASSET_TRANSFER_WAIT_FOR_NEW_OWNER {
		// args[0] - Asset
		// args[1] - New Owner Signature
		// args[2] - New Owner Public Key

		return t.assetTransferDeclineByNewOwner(stub, args)
	}

	return nil, nil
}

func (t *DNSChaincode) assetTransferAcceptByNewOwner(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args) < 4 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	// args[0] - Asset
	// args[1] - New Owner Signature
	// args[2] - New Owner Public Key
	// args[3] - New Owner Asset Signature

	if !t.checkAssetOwnership(stub, args[0], args[1], args[2], args, true) {
		return nil, errors.New("Invalid ownership")
	}

	var assetRow, tradeRow shim.Row
	var convertedValue float64
	var convertedTradeAmount, convertedAssetAmount int
	var err error

	assetRow, err = stub.GetRow(REGISTERED_ASSETS_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: args[0]}}})
	if err != nil || len(assetRow.Columns) == 0 {
		return nil, errors.New("Invalid asset name")
	}

	var pubkey = assetRow.Columns[ASSET_PUBLIC_KEY_COLUMN].GetString_()

	convertedAssetAmount, err = strconv.Atoi(assetRow.Columns[ASSET_AMOUNT_COLUMN].GetString_())
	if err != nil {
		return nil, fmt.Errorf("Failed to convert: %s", err)
	}

	tradeRow, err = stub.GetRow(OPEN_TRADE_TABLE, []shim.Column{
		{Value: &shim.Column_String_{String_: strings.TrimSpace(pubkey)}},
		{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}},
	})
	if err != nil && len(tradeRow.Columns) != 0 {
		return nil, fmt.Errorf("Row not found: %s", err)
	}

	var newAssetName string = tradeRow.Columns[TRADE_NEW_ASSET_NAME_COLUMN].GetString_()

	convertedValue, err = strconv.ParseFloat(tradeRow.Columns[TRADE_VALUE_COLUMN].GetString_(), 64)
	if err != nil {
		return nil, fmt.Errorf("Failed to convert: %s", err)
	}

	convertedTradeAmount, err = strconv.Atoi(tradeRow.Columns[TRADE_AMOUNT_COLUMN].GetString_())
	if err != nil {
		return nil, fmt.Errorf("Failed to convert: %s", err)
	}

	err = t.profileCheckCredits(stub, strings.TrimSpace(strings.Replace(args[2], " ", "", -1)), convertedValue)
	if err != nil {
		return nil, err
	}

	err = stub.DeleteRow(OPEN_TRADE_TABLE, []shim.Column{
		{Value: &shim.Column_String_{String_: strings.TrimSpace(pubkey)}},
		{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}},
	})
	if err != nil {
		return nil, fmt.Errorf("Error deleting row for the trade: %s", err)
	}

	err = stub.DeleteRow(OPEN_TRADE_TABLE, []shim.Column{
		{Value: &shim.Column_String_{String_: strings.TrimSpace(args[2])}},
		{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}},
	})
	if err != nil {
		return nil, fmt.Errorf("Error deleting row for the trade: %s", err)
	}

	if convertedTradeAmount >= convertedAssetAmount {
		err = t.assetUpdateOwner(stub, strings.TrimSpace(args[0]), strings.TrimSpace(args[2]), strings.TrimSpace(args[3]), convertedValue)
		if err != nil {
			return nil, fmt.Errorf("Error updating owner: %s", err)
		}
	} else {
		err = t.assetSplitOwner(stub, strings.TrimSpace(args[0]), strings.TrimSpace(newAssetName), strings.TrimSpace(args[2]), strings.TrimSpace(args[3]), convertedTradeAmount, convertedAssetAmount, convertedValue)
		if err != nil {
			return nil, fmt.Errorf("Error updating owner: %s", err)
		}
	}

	err = t.assetUpdateLock(stub, args[0], false)
	if err != nil {
		return nil, fmt.Errorf("Error updating asset: %s", err)
	}

	err = t.updateStats(stub, OPEN_ASSET_TRADE_STATS_NAME, -1)
	if err != nil {
		return nil, fmt.Errorf("Error updating stats: %s", err)
	}

	err = t.updateStats(stub, ACCEPTED_ASSET_TRADE_STATS_NAME, 1)
	if err != nil {
		return nil, fmt.Errorf("Error updating stats: %s", err)
	}

	err = t.updateStats(stub, CLOSE_ASSET_TRADE_STATS_NAME, 1)
	if err != nil {
		return nil, fmt.Errorf("Error updating stats: %s", err)
	}

	return nil, nil
}

func (t *DNSChaincode) assetTransferDeclineByNewOwner(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args) < 3 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	// args[0] - Asset
	// args[1] - New Owner Signature
	// args[2] - New Owner Public Key

	if !t.checkAssetOwnership(stub, args[0], args[1], args[2], args, true) {
		return nil, errors.New("Invalid ownership")
	}

	var assetRow, tradeRow shim.Row
	var err error

	assetRow, err = stub.GetRow(REGISTERED_ASSETS_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: args[0]}}})
	if err != nil || len(assetRow.Columns) == 0 {
		return nil, errors.New("Invalid asset name")
	}

	var pubkey = assetRow.Columns[ASSET_PUBLIC_KEY_COLUMN].GetString_()

	tradeRow, err = stub.GetRow(OPEN_TRADE_TABLE, []shim.Column{
		{Value: &shim.Column_String_{String_: strings.TrimSpace(pubkey)}},
		{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}},
	})
	if err != nil && len(tradeRow.Columns) != 0 {
		return nil, fmt.Errorf("Row not found: %s", err)
	}

	err = stub.DeleteRow(OPEN_TRADE_TABLE, []shim.Column{
		{Value: &shim.Column_String_{String_: strings.TrimSpace(pubkey)}},
		{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}},
	})
	if err != nil {
		return nil, fmt.Errorf("Error deleting row for the trade: %s", err)
	}

	err = stub.DeleteRow(OPEN_TRADE_TABLE, []shim.Column{
		{Value: &shim.Column_String_{String_: strings.TrimSpace(args[2])}},
		{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}},
	})
	if err != nil {
		return nil, fmt.Errorf("Error deleting row for the trade: %s", err)
	}

	err = t.assetUpdateLock(stub, strings.TrimSpace(args[0]), false)
	if err != nil {
		return nil, fmt.Errorf("Error updating asset: %s", err)
	}

	err = t.updateStats(stub, OPEN_ASSET_TRADE_STATS_NAME, -1)
	if err != nil {
		return nil, fmt.Errorf("Error updating stats: %s", err)
	}

	err = t.updateStats(stub, CLOSE_ASSET_TRADE_STATS_NAME, 1)
	if err != nil {
		return nil, fmt.Errorf("Error updating stats: %s", err)
	}

	return nil, nil
}

func (t *DNSChaincode) assetTransferAcceptByOwner(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if !t.checkAssetOwnership(stub, args[0], args[1], "", args, false) {
		return nil, errors.New("Invalid ownership")
	}

	var assetRow, tradeRow shim.Row
	var err error

	assetRow, err = stub.GetRow(REGISTERED_ASSETS_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: args[0]}}})
	if err != nil || len(assetRow.Columns) == 0 {
		return nil, errors.New("Invalid asset name")
	}

	tradeRow, err = stub.GetRow(OPEN_TRADE_TABLE, []shim.Column{
		{Value: &shim.Column_String_{String_: assetRow.Columns[ASSET_PUBLIC_KEY_COLUMN].GetString_()}},
		{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}},
	})
	if err != nil && len(tradeRow.Columns) != 0 {
		return nil, fmt.Errorf("Row not found: %s", err)
	}

	_, err = stub.ReplaceRow(OPEN_TRADE_TABLE, shim.Row{
		[]*shim.Column{
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_USER_ID_A_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_PUBLIC_KEY_A_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_ASSET_NAME_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: ASSET_TRANSFER_WAIT_FOR_NEW_OWNER}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_USER_ID_B_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_PUBLIC_KEY_B_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_CURRENT_OWNER_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_NEW_ASSET_NAME_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_AMOUNT_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_STARTING_VALUE_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_VALUE_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_TIME_SPAN_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_SIGNATURE_COLUMN].GetString_()}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Error updating row for the trade: %s", err)
	}

	_, err = stub.ReplaceRow(OPEN_TRADE_TABLE, shim.Row{
		[]*shim.Column{
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_USER_ID_B_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_PUBLIC_KEY_B_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_ASSET_NAME_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: ASSET_TRANSFER_WAIT_FOR_NEW_OWNER}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_USER_ID_A_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_PUBLIC_KEY_A_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_CURRENT_OWNER_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_NEW_ASSET_NAME_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_AMOUNT_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_STARTING_VALUE_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_VALUE_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_TIME_SPAN_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_SIGNATURE_COLUMN].GetString_()}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Error updating row for the trade: %s", err)
	}

	return nil, nil
}

func (t *DNSChaincode) assetTransferDeclineByOwner(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if !t.checkAssetOwnership(stub, args[0], args[1], "", args, false) {
		return nil, errors.New("Invalid ownership")
	}

	var assetRow, tradeRow shim.Row
	var err error

	assetRow, err = stub.GetRow(REGISTERED_ASSETS_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: args[0]}}})
	if err != nil || len(assetRow.Columns) == 0 {
		return nil, errors.New("Invalid asset name")
	}

	tradeRow, err = stub.GetRow(OPEN_TRADE_TABLE, []shim.Column{
		{Value: &shim.Column_String_{String_: assetRow.Columns[ASSET_PUBLIC_KEY_COLUMN].GetString_()}},
		{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}},
	})
	if err != nil && len(tradeRow.Columns) != 0 {
		return nil, fmt.Errorf("Row not found: %s", err)
	}

	_, err = stub.ReplaceRow(OPEN_TRADE_TABLE, shim.Row{
		[]*shim.Column{
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_USER_ID_A_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_PUBLIC_KEY_A_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_ASSET_NAME_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: ASSET_TRANSFER_WAIT_FOR_NEW_OWNER}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_USER_ID_B_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_PUBLIC_KEY_B_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_CURRENT_OWNER_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_NEW_ASSET_NAME_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_AMOUNT_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_STARTING_VALUE_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_STARTING_VALUE_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_TIME_SPAN_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_SIGNATURE_COLUMN].GetString_()}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Error updating row for the trade: %s", err)
	}

	_, err = stub.ReplaceRow(OPEN_TRADE_TABLE, shim.Row{
		[]*shim.Column{
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_USER_ID_B_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_PUBLIC_KEY_B_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_ASSET_NAME_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: ASSET_TRANSFER_WAIT_FOR_NEW_OWNER}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_USER_ID_A_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_PUBLIC_KEY_A_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_CURRENT_OWNER_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_NEW_ASSET_NAME_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_AMOUNT_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_STARTING_VALUE_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_STARTING_VALUE_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_TIME_SPAN_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_SIGNATURE_COLUMN].GetString_()}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Error updating row for the trade: %s", err)
	}

	return nil, nil
}

func (t *DNSChaincode) assetTransferCancelByOwner(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2")
	}

	// args[0] - Asset
	// args[1] - Owner Signature

	if !t.checkAssetOwnership(stub, args[0], args[1], "", args, false) {
		return nil, errors.New("Invalid ownership")
	}

	var assetRow, tradeRow shim.Row
	var err error

	assetRow, err = stub.GetRow(REGISTERED_ASSETS_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: args[0]}}})
	if err != nil || len(assetRow.Columns) == 0 {
		return nil, errors.New("Invalid asset name")
	}

	var pubkeyA = assetRow.Columns[ASSET_PUBLIC_KEY_COLUMN].GetString_()

	tradeRow, err = stub.GetRow(OPEN_TRADE_TABLE, []shim.Column{
		{Value: &shim.Column_String_{String_: strings.TrimSpace(pubkeyA)}},
		{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}},
	})
	if err != nil && len(tradeRow.Columns) != 0 {
		return nil, fmt.Errorf("Row not found: %s", err)
	}

	var pubkeyB = tradeRow.Columns[TRADE_PUBLIC_KEY_B_COLUMN].GetString_()

	err = stub.DeleteRow(OPEN_TRADE_TABLE, []shim.Column{
		{Value: &shim.Column_String_{String_: strings.TrimSpace(pubkeyA)}},
		{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}},
	})
	if err != nil {
		return nil, fmt.Errorf("Error deleting row for the trade: %s", err)
	}

	err = stub.DeleteRow(OPEN_TRADE_TABLE, []shim.Column{
		{Value: &shim.Column_String_{String_: strings.TrimSpace(pubkeyB)}},
		{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}},
	})
	if err != nil {
		return nil, fmt.Errorf("Error deleting row for the trade: %s", err)
	}

	err = t.assetUpdateLock(stub, strings.TrimSpace(args[0]), false)
	if err != nil {
		return nil, fmt.Errorf("Error updating asset: %s", err)
	}

	err = t.updateStats(stub, OPEN_ASSET_TRADE_STATS_NAME, -1)
	if err != nil {
		return nil, fmt.Errorf("Error updating stats: %s", err)
	}

	err = t.updateStats(stub, CLOSE_ASSET_TRADE_STATS_NAME, 1)
	if err != nil {
		return nil, fmt.Errorf("Error updating stats: %s", err)
	}

	return nil, nil
}

func (t *DNSChaincode) assetTransferMakeOffer(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args) < 4 {
		return nil, errors.New("Incorrect number of arguments. Expecting 4")
	}

	// args[0] - Asset
	// args[1] - New Owner Signature
	// args[2] - New Owner Public Key
	// args[3] - New Credit Amount

	if !t.checkAssetOwnership(stub, args[0], args[1], args[2], args, true) {
		return nil, errors.New("Invalid ownership")
	}

	var tradeRow shim.Row
	var err error

	tradeRow, err = stub.GetRow(OPEN_TRADE_TABLE, []shim.Column{
		{Value: &shim.Column_String_{String_: strings.TrimSpace(args[2])}},
		{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}},
	})
	if err != nil && len(tradeRow.Columns) != 0 {
		return nil, fmt.Errorf("Row not found: %s", err)
	}

	_, err = stub.ReplaceRow(OPEN_TRADE_TABLE, shim.Row{
		[]*shim.Column{
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_USER_ID_A_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_PUBLIC_KEY_A_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_ASSET_NAME_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: ASSET_TRANSFER_WAIT_FOR_OWNER}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_USER_ID_B_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_PUBLIC_KEY_B_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_CURRENT_OWNER_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_NEW_ASSET_NAME_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_AMOUNT_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_STARTING_VALUE_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: strings.TrimSpace(args[3])}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_TIME_SPAN_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_SIGNATURE_COLUMN].GetString_()}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Error updating row for the trade: %s", err)
	}

	_, err = stub.ReplaceRow(OPEN_TRADE_TABLE, shim.Row{
		[]*shim.Column{
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_USER_ID_B_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_PUBLIC_KEY_B_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_ASSET_NAME_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: ASSET_TRANSFER_WAIT_FOR_OWNER}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_USER_ID_A_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_PUBLIC_KEY_A_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_CURRENT_OWNER_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_NEW_ASSET_NAME_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_AMOUNT_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_STARTING_VALUE_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: strings.TrimSpace(args[3])}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_TIME_SPAN_COLUMN].GetString_()}},
			{Value: &shim.Column_String_{String_: tradeRow.Columns[TRADE_SIGNATURE_COLUMN].GetString_()}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Error updating row for the trade: %s", err)
	}

	err = t.updateStats(stub, MAKE_OFFER_ASSET_TRADE_STATS_NAME, 1)
	if err != nil {
		return nil, fmt.Errorf("Error updating stats: %s", err)
	}

	return nil, nil
}

func (t *DNSChaincode) assetUpdateOwner(stub *shim.ChaincodeStub, assetName string, publicKey string, signature string, assetCost float64) error {
	var assetsRow shim.Row
	var rowAdded bool
	var err error

	assetsRow, err = stub.GetRow(REGISTERED_ASSETS_TABLE, []shim.Column{
		{Value: &shim.Column_String_{String_: assetName}},
	})

	if err == nil && len(assetsRow.Columns) != 0 {
		var key string = t.readStringSafe(assetsRow.Columns[ASSET_PUBLIC_KEY_COLUMN])

		_, err = stub.ReplaceRow(REGISTERED_ASSETS_TABLE, shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_NAME_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_DESCRIPTION_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_REGISTERED_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_EXPIRES_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_TIME_SPAN_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_LAST_UPDATED_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_AMOUNT_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_VALUE_COLUMN].GetString_()}},
				{Value: &shim.Column_Bool{Bool: assetsRow.Columns[ASSET_LOCKED_COLUMN].GetBool()}},
				{Value: &shim.Column_String_{String_: publicKey}},
				{Value: &shim.Column_String_{String_: signature}},
			},
		})
		if err != nil {
			return fmt.Errorf("Error updating row for the asset: %s", err)
		}

		err = stub.DeleteRow(REGISTERED_ASSETS_ID_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: key}}, {Value: &shim.Column_String_{String_: strings.TrimSpace(assetName)}}})
		if err != nil {
			return fmt.Errorf("Error deleting row for the asset: %s", err)
		}

		rowAdded, err = stub.InsertRow(REGISTERED_ASSETS_ID_TABLE, shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: publicKey}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_NAME_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_DESCRIPTION_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_REGISTERED_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_EXPIRES_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_TIME_SPAN_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_LAST_UPDATED_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_AMOUNT_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_VALUE_COLUMN].GetString_()}},
				{Value: &shim.Column_Bool{Bool: assetsRow.Columns[ASSET_LOCKED_COLUMN].GetBool()}},
				{Value: &shim.Column_String_{String_: signature}},
			},
		})
		if err != nil || !rowAdded {
			return errors.New(fmt.Sprintf("Error creating row: %s", err))
		}

		err = t.profileUpdateCredits(stub, publicKey, assetCost)
		if err != nil {
			return fmt.Errorf("Error updating credits: %s", err)
		}

		err = t.profileUpdateAssets(stub, key, assetName, PROFILE_ASSETS_DELETE)
		if err != nil {
			return fmt.Errorf("Error updating assets: %s", err)
		}

		err = t.profileUpdateAssets(stub, publicKey, assetName, PROFILE_ASSETS_ADD)
		if err != nil {
			return fmt.Errorf("Error updating assets: %s", err)
		}
	}

	return nil
}

func (t *DNSChaincode) assetSplitOwner(stub *shim.ChaincodeStub, assetName string, newAssetName string, publicKey string, signature string, convertedTradeAmount int, convertedAssetAmount int, assetCost float64) error {
	var assetsRow shim.Row
	var rowAdded bool
	var err error

	assetsRow, err = stub.GetRow(REGISTERED_ASSETS_TABLE, []shim.Column{
		{Value: &shim.Column_String_{String_: assetName}},
	})

	if err == nil && len(assetsRow.Columns) != 0 {
		_, err = stub.ReplaceRow(REGISTERED_ASSETS_TABLE, shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_NAME_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_DESCRIPTION_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_REGISTERED_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_EXPIRES_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_TIME_SPAN_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: time.Now().Format(time.RFC822Z)}},
				{Value: &shim.Column_String_{String_: strconv.Itoa(convertedAssetAmount - convertedTradeAmount)}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_VALUE_COLUMN].GetString_()}},
				{Value: &shim.Column_Bool{Bool: assetsRow.Columns[ASSET_LOCKED_COLUMN].GetBool()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_PUBLIC_KEY_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_SIGNATURE_COLUMN].GetString_()}},
			},
		})
		if err != nil {
			return fmt.Errorf("Error updating row for the asset: %s", err)
		}

		_, err = stub.ReplaceRow(REGISTERED_ASSETS_ID_TABLE, shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_PUBLIC_KEY_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_NAME_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_DESCRIPTION_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_REGISTERED_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_EXPIRES_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_TIME_SPAN_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: time.Now().Format(time.RFC822Z)}},
				{Value: &shim.Column_String_{String_: strconv.Itoa(convertedAssetAmount - convertedTradeAmount)}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_VALUE_COLUMN].GetString_()}},
				{Value: &shim.Column_Bool{Bool: assetsRow.Columns[ASSET_LOCKED_COLUMN].GetBool()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_SIGNATURE_COLUMN].GetString_()}},
			},
		})
		if err != nil {
			return fmt.Errorf("Error updating row for the asset: %s", err)
		}

		rowAdded, err = stub.InsertRow(REGISTERED_ASSETS_TABLE, shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: newAssetName}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_DESCRIPTION_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_REGISTERED_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_EXPIRES_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_TIME_SPAN_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: time.Now().Format(time.RFC822Z)}},
				{Value: &shim.Column_String_{String_: strconv.Itoa(convertedTradeAmount)}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_VALUE_COLUMN].GetString_()}},
				{Value: &shim.Column_Bool{Bool: false}},
				{Value: &shim.Column_String_{String_: publicKey}},
				{Value: &shim.Column_String_{String_: signature}},
			},
		})
		if err != nil || !rowAdded {
			return errors.New(fmt.Sprintf("Error creating row: %s", err))
		}

		rowAdded, err = stub.InsertRow(REGISTERED_ASSETS_ID_TABLE, shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: publicKey}},
				{Value: &shim.Column_String_{String_: newAssetName}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_DESCRIPTION_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_REGISTERED_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_EXPIRES_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_TIME_SPAN_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: time.Now().Format(time.RFC822Z)}},
				{Value: &shim.Column_String_{String_: strconv.Itoa(convertedTradeAmount)}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_VALUE_COLUMN].GetString_()}},
				{Value: &shim.Column_Bool{Bool: false}},
				{Value: &shim.Column_String_{String_: signature}},
			},
		})
		if err != nil || !rowAdded {
			return errors.New(fmt.Sprintf("Error creating row: %s", err))
		}

		err = t.profileUpdateCredits(stub, publicKey, assetCost)
		if err != nil {
			return fmt.Errorf("Error updating credits: %s", err)
		}

		err = t.profileUpdateAssets(stub, publicKey, newAssetName, PROFILE_ASSETS_ADD)
		if err != nil {
			return fmt.Errorf("Error updating assets: %s", err)
		}
	}

	return nil
}

func (t *DNSChaincode) assetUpdateLock(stub *shim.ChaincodeStub, assetName string, newLockStatus bool) error {
	var assetsRow shim.Row
	var err error

	assetsRow, err = stub.GetRow(REGISTERED_ASSETS_TABLE, []shim.Column{
		{Value: &shim.Column_String_{String_: assetName}},
	})

	if err == nil && len(assetsRow.Columns) != 0 {
		_, err = stub.ReplaceRow(REGISTERED_ASSETS_TABLE, shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_NAME_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_DESCRIPTION_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_REGISTERED_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_EXPIRES_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_TIME_SPAN_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_LAST_UPDATED_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_AMOUNT_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_VALUE_COLUMN].GetString_()}},
				{Value: &shim.Column_Bool{Bool: newLockStatus}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_PUBLIC_KEY_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_SIGNATURE_COLUMN].GetString_()}},
			},
		})
		if err != nil {
			return fmt.Errorf("Error updating row for the asset: %s", err)
		}

		_, err = stub.ReplaceRow(REGISTERED_ASSETS_ID_TABLE, shim.Row{
			[]*shim.Column{
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_PUBLIC_KEY_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_NAME_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_DESCRIPTION_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_REGISTERED_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_EXPIRES_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_TIME_SPAN_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_LAST_UPDATED_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_AMOUNT_COLUMN].GetString_()}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_VALUE_COLUMN].GetString_()}},
				{Value: &shim.Column_Bool{Bool: newLockStatus}},
				{Value: &shim.Column_String_{String_: assetsRow.Columns[ASSET_SIGNATURE_COLUMN].GetString_()}},
			},
		})
		if err != nil {
			return fmt.Errorf("Error updating row for the asset: %s", err)
		}
	}

	return nil
}

func (t *DNSChaincode) assetRenew(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args) < 6 {
		return nil, errors.New("Incorrect number of arguments. Expecting 6")
	}

	// args[0] - Asset
	// args[1] - Owner Signature
	// args[2] - Length
	// args[3] - True/False - Use length for the cost
	// args[4] - True/False - Does it cost to renew?
	// args[5] - Rate for the cost of the length.

	if !t.checkAssetOwnership(stub, args[0], args[1], "", args, false) {
		return nil, errors.New("Invalid ownership")
	}

	var newExpiresTime, newUpdatedTime string
	var expiresTime time.Time
	var row shim.Row
	var convertedLength int
	var err error

	convertedLength, err = strconv.Atoi(args[2])
	if err != nil {
		return nil, errors.New("Invalid length")
	}

	if convertedLength <= 0 {
		return nil, errors.New("The length must be greater than zero")
	}

	row, err = stub.GetRow(REGISTERED_ASSETS_TABLE, []shim.Column{
		{Value: &shim.Column_String_{String_: args[0]}},
	})

	if err == nil && len(row.Columns) != 0 {
		var assetCost, convertedValue, convertedAmount, convertedRate float64
		var convertedTimeSpan int
		var checkForLength, checkIfCost bool

		convertedValue, err = strconv.ParseFloat(row.Columns[ASSET_VALUE_COLUMN].GetString_(), 64)
		if err != nil {
			return nil, fmt.Errorf("Failed to convert: %s", err)
		}

		convertedAmount, err = strconv.ParseFloat(row.Columns[ASSET_AMOUNT_COLUMN].GetString_(), 64)
		if err != nil {
			return nil, fmt.Errorf("Failed to convert: %s", err)
		}

		convertedTimeSpan, err = strconv.Atoi(row.Columns[ASSET_TIME_SPAN_COLUMN].GetString_())
		if err != nil {
			return nil, errors.New("Invalid time span")
		}

		checkIfCost, err = strconv.ParseBool(args[4])
		if err == nil {
			if checkIfCost {
				convertedRate, err = strconv.ParseFloat(args[5], 64)
				if err != nil {
					return nil, fmt.Errorf("Failed to convert: %s", err)
				}

				if convertedRate <= 0 {
					convertedRate = 1.0
				}

				checkForLength, err = strconv.ParseBool(args[3])
				if err == nil {
					if checkForLength {
						assetCost = convertedAmount * convertedValue * float64(convertedLength) * convertedRate
					} else {
						assetCost = convertedAmount * convertedValue
					}
				} else {
					return nil, err
				}
			} else {
				assetCost = 0.0
			}
		} else {
			return nil, err
		}

		err = t.profileCheckCredits(stub, strings.TrimSpace(strings.Replace(row.Columns[ASSET_PUBLIC_KEY_COLUMN].GetString_(), " ", "", -1)), assetCost)
		if err != nil {
			return nil, err
		}

		if !row.Columns[ASSET_LOCKED_COLUMN].GetBool() {
			expiresTime, err = time.Parse(time.RFC822Z, row.Columns[ASSET_EXPIRES_COLUMN].GetString_())

			if err == nil {
				newExpiresTime = expiresTime.Add(time.Hour * 24 * time.Duration(convertedLength)).Format(time.RFC822Z)
				newUpdatedTime = time.Now().Format(time.RFC822Z)

				_, err = stub.ReplaceRow(REGISTERED_ASSETS_TABLE, shim.Row{
					[]*shim.Column{
						{Value: &shim.Column_String_{String_: row.Columns[ASSET_NAME_COLUMN].GetString_()}},
						{Value: &shim.Column_String_{String_: row.Columns[ASSET_DESCRIPTION_COLUMN].GetString_()}},
						{Value: &shim.Column_String_{String_: row.Columns[ASSET_REGISTERED_COLUMN].GetString_()}},
						{Value: &shim.Column_String_{String_: newExpiresTime}},
						{Value: &shim.Column_String_{String_: strconv.Itoa(convertedTimeSpan + convertedLength)}},
						{Value: &shim.Column_String_{String_: newUpdatedTime}},
						{Value: &shim.Column_String_{String_: row.Columns[ASSET_AMOUNT_COLUMN].GetString_()}},
						{Value: &shim.Column_String_{String_: row.Columns[ASSET_VALUE_COLUMN].GetString_()}},
						{Value: &shim.Column_Bool{Bool: row.Columns[ASSET_LOCKED_COLUMN].GetBool()}},
						{Value: &shim.Column_String_{String_: row.Columns[ASSET_PUBLIC_KEY_COLUMN].GetString_()}},
						{Value: &shim.Column_String_{String_: row.Columns[ASSET_SIGNATURE_COLUMN].GetString_()}},
					},
				})

				if err != nil {
					return nil, fmt.Errorf("Error updating row for the asset: %s", err)
				}

				_, err = stub.ReplaceRow(REGISTERED_ASSETS_ID_TABLE, shim.Row{
					[]*shim.Column{
						{Value: &shim.Column_String_{String_: row.Columns[ASSET_PUBLIC_KEY_COLUMN].GetString_()}},
						{Value: &shim.Column_String_{String_: row.Columns[ASSET_NAME_COLUMN].GetString_()}},
						{Value: &shim.Column_String_{String_: row.Columns[ASSET_DESCRIPTION_COLUMN].GetString_()}},
						{Value: &shim.Column_String_{String_: row.Columns[ASSET_REGISTERED_COLUMN].GetString_()}},
						{Value: &shim.Column_String_{String_: newExpiresTime}},
						{Value: &shim.Column_String_{String_: strconv.Itoa(convertedTimeSpan + convertedLength)}},
						{Value: &shim.Column_String_{String_: newUpdatedTime}},
						{Value: &shim.Column_String_{String_: row.Columns[ASSET_AMOUNT_COLUMN].GetString_()}},
						{Value: &shim.Column_String_{String_: row.Columns[ASSET_VALUE_COLUMN].GetString_()}},
						{Value: &shim.Column_Bool{Bool: row.Columns[ASSET_LOCKED_COLUMN].GetBool()}},
						{Value: &shim.Column_String_{String_: row.Columns[ASSET_SIGNATURE_COLUMN].GetString_()}},
					},
				})

				if err != nil {
					return nil, fmt.Errorf("Error updating row for the asset: %s", err)
				}

				err = t.updateStats(stub, TOTAL_RENEW_ASSETS_STATS_NAME, 1)
				if err != nil {
					return nil, fmt.Errorf("Error updating stats: %s", err)
				}

				err = t.profileUpdateCredits(stub, row.Columns[ASSET_PUBLIC_KEY_COLUMN].GetString_(), assetCost)
				if err != nil {
					return nil, fmt.Errorf("Error updating credits: %s", err)
				}
			}
		} else {
			return nil, errors.New("Asset is locked")
		}
	} else {
		return nil, err
	}

	return nil, nil
}

func (t *DNSChaincode) assetDelete(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args) < 4 {
		return nil, errors.New("Incorrect number of arguments. Expecting 4")
	}

	// args[0] - Asset
	// args[1] - Owner Signature
	// args[2] - True/False to give credits back when an asset is deleting
	// args[3] - How much of the value is returned

	if !t.checkAssetOwnership(stub, args[0], args[1], "", args, false) {
		return nil, errors.New("Invalid ownership")
	}

	var row shim.Row
	var checkIfReturn bool
	var convertedValue, convertedAmount, convertedReturnRate float64
	var expiresTime time.Time
	var err error

	row, err = stub.GetRow(REGISTERED_ASSETS_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}}})
	if err != nil || len(row.Columns) == 0 {
		return nil, fmt.Errorf("Failed to get state for "+args[0]+": %s", err)
	}

	if !row.Columns[ASSET_LOCKED_COLUMN].GetBool() {
		var key string = t.readStringSafe(row.Columns[ASSET_PUBLIC_KEY_COLUMN])

		checkIfReturn, err = strconv.ParseBool(args[2])
		if err != nil {
			return nil, err
		}

		if checkIfReturn {
			convertedReturnRate, err = strconv.ParseFloat(args[3], 64)
			if err != nil {
				return nil, fmt.Errorf("Failed to convert: %s", err)
			}

			convertedValue, err = strconv.ParseFloat(row.Columns[ASSET_VALUE_COLUMN].GetString_(), 64)
			if err != nil {
				return nil, fmt.Errorf("Failed to convert: %s", err)
			}

			convertedAmount, err = strconv.ParseFloat(row.Columns[ASSET_AMOUNT_COLUMN].GetString_(), 64)
			if err != nil {
				return nil, fmt.Errorf("Failed to convert: %s", err)
			}

			expiresTime, err = time.Parse(time.RFC822Z, row.Columns[ASSET_EXPIRES_COLUMN].GetString_())
			if err != nil {
				return nil, fmt.Errorf("Failed to convert: %s", err)
			}
		}

		err = stub.DeleteRow(REGISTERED_ASSETS_ID_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: key}}, {Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}}})
		if err != nil {
			return nil, fmt.Errorf("Error deleting row for the asset: %s", err)
		}

		err = stub.DeleteRow(REGISTERED_ASSETS_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: args[0]}}})
		if err != nil {
			return nil, fmt.Errorf("Error deleting row for the asset: %s", err)
		}

		err = t.profileUpdateAssets(stub, key, args[0], PROFILE_ASSETS_DELETE)
		if err != nil {
			return nil, fmt.Errorf("Error updating assets: %s", err)
		}

		err = t.updateStats(stub, CURRENT_ASSETS_STATS_NAME, -1)
		if err != nil {
			return nil, fmt.Errorf("Error updating stats: %s", err)
		}

		if checkIfReturn && time.Now().Before(expiresTime) {
			err = t.profileUpdateCredits(stub, key, convertedAmount*convertedValue*convertedReturnRate*-1.0)
			if err != nil {
				return nil, fmt.Errorf("Error updating credits: %s", err)
			}
		}
	} else {
		return nil, errors.New("Asset is locked")
	}

	return nil, nil
}

// Query is our entry point for queries
func (t *DNSChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

	// Handle different functions
	switch(function) {

	case "AllRegisteredDomains":
		if len(args) != 0 {
			return nil, errors.New("Incorrect number of arguments. Expecting 0")
		}

		data, r_err = t.getDomains(stub, args)
		if r_err != nil {
			return nil, errors.New("{\"error\":\"" + r_err.Error() + "\"}")
		}
		break

	case "IPToName":
		if len(args) != 1 {
			return nil, errors.New("Incorrect number of arguments. Expecting 1")
		}

		data, r_err = t.getIPToName(stub, args)
		if r_err != nil {
			return nil, errors.New("{\"error\":\"" + r_err.Error() + "\"}")
		}
		break

	case "NameToIP":
		if len(args) != 1 {
			return nil, errors.New("Incorrect number of arguments. Expecting 1")
		}

		data, r_err = t.getNameToIP(stub, args)
		if r_err != nil {
			return nil, errors.New("{\"error\":\"" + r_err.Error() + "\"}")
		}
		break

	case "UserBids":
		if len(args) != 1 {
			return nil, errors.New("Incorrect number of arguments. Expecting 1")
		}

		data, r_err = t.getUserBids(stub, args)
		if r_err != nil {
			return nil, errors.New("{\"error\":\"" + r_err.Error() + "\"}")
		}
		break

	case "ReceivedBids":
		if len(args) < 1 {
			return nil, errors.New("Incorrect number of arguments. Expecting 1")
		}

		data, r_err = t.getReceivedBids(stub, args)
		if r_err != nil {
			return nil, errors.New("{\"error\":\"" + r_err.Error() + "\"}")
		}
		break

	case default:
		fmt.Println("query did not find func: " + function)
		return nil, errors.New("Received unknown function query")
	}
	var converted []byte
	var converted_err error

	converted, converted_err = json.Marshal(data)
	if converted_err != nil {
		return nil, errors.New("{\"error\":\"" + converted_err.Error() + "\"}")
	}

	return converted, nil
}

func (t *DNSChaincode) convertToUserAssetRecord(stub *shim.ChaincodeStub, args []string) (AssetRecordList, error) {
	if len(args) != 1 {
		return AssetRecordList{}, errors.New("Incorrect number of arguments. Expecting 1")
	}

	var row shim.Row
	var qRow <-chan shim.Row
	var err error

	row, err = stub.GetRow(REGISTERED_PROFILE_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}}})
	if err != nil || len(row.Columns) == 0 {
		return AssetRecordList{}, fmt.Errorf("Failed to get state for "+args[0]+": %s", err)
	}

	qRow, err = stub.GetRows(REGISTERED_ASSETS_ID_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: strings.TrimSpace(t.readStringSafe(row.Columns[PROFILE_PUBLIC_KEY_COLUMN]))}}})
	if err != nil {
		return AssetRecordList{}, fmt.Errorf("Failed to get state for "+args[0]+": %s", err)
	}

	var assetRecords []AssetRecord

	for {
		select {
		case currRow, ok := <-qRow:
			if !ok {
				qRow = nil
			} else {
				if len(currRow.Columns) != 0 {
					expiresTime, err := time.Parse(time.RFC822Z, currRow.Columns[ASSET_ID_EXPIRES_COLUMN].GetString_())

					var assetValue string
					var isExpired bool
					if err == nil {
						if time.Now().Before(expiresTime) {
							isExpired = false
							assetValue = t.readStringSafe(currRow.Columns[ASSET_ID_VALUE_COLUMN])
						} else {
							isExpired = true
							assetValue = "0.00"
						}
					}

					assetRecords = append(assetRecords, AssetRecord{
						Name:        t.readStringSafe(currRow.Columns[ASSET_ID_NAME_COLUMN]),
						Description: t.readStringSafe(currRow.Columns[ASSET_ID_DESCRIPTION_COLUMN]),
						Registered:  t.readStringSafe(currRow.Columns[ASSET_ID_REGISTERED_COLUMN]),
						Expires:     t.readStringSafe(currRow.Columns[ASSET_ID_EXPIRES_COLUMN]),
						Expired:     isExpired,
						TimeSpan:    t.readStringSafe(currRow.Columns[ASSET_ID_TIME_SPAN_COLUMN]),
						LastUpdated: t.readStringSafe(currRow.Columns[ASSET_ID_LAST_UPDATED_COLUMN]),
						Amount:      t.readStringSafe(currRow.Columns[ASSET_ID_AMOUNT_COLUMN]),
						Value:       assetValue,
						Locked:      t.readBoolSafe(currRow.Columns[ASSET_ID_LOCKED_COLUMN]),
						PublicKey:   t.readStringSafe(currRow.Columns[ASSET_ID_PUBLIC_KEY_COLUMN]),
						Signature:   t.readStringSafe(currRow.Columns[ASSET_ID_SIGNATURE_COLUMN]),
					})
				}
			}
		}
		if qRow == nil {
			break
		}
	}

	return AssetRecordList{AssetList: assetRecords, ProfileCredits: t.readStringSafe(row.Columns[PROFILE_CREDITS_COLUMN])}, nil
}

func (t *DNSChaincode) convertToAssetRecord(stub *shim.ChaincodeStub, args []string) (AssetRecord, error) {
	if len(args) != 1 {
		return AssetRecord{}, errors.New("Incorrect number of arguments. Expecting 1")
	}

	row, r_err := stub.GetRow(REGISTERED_ASSETS_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}}})

	if r_err != nil || len(row.Columns) == 0 {
		return AssetRecord{}, fmt.Errorf("Failed to get state for " + args[0])
	}

	expiresTime, err := time.Parse(time.RFC822Z, row.Columns[ASSET_EXPIRES_COLUMN].GetString_())

	var assetValue string
	var isExpired bool
	if err == nil {
		if time.Now().Before(expiresTime) {
			isExpired = false
			assetValue = t.readStringSafe(row.Columns[ASSET_VALUE_COLUMN])
		} else {
			isExpired = true
			assetValue = "0.00"
		}
	}

	return AssetRecord{
		Name:        t.readStringSafe(row.Columns[ASSET_NAME_COLUMN]),
		Description: t.readStringSafe(row.Columns[ASSET_DESCRIPTION_COLUMN]),
		Registered:  t.readStringSafe(row.Columns[ASSET_REGISTERED_COLUMN]),
		Expires:     t.readStringSafe(row.Columns[ASSET_EXPIRES_COLUMN]),
		Expired:     isExpired,
		TimeSpan:    t.readStringSafe(row.Columns[ASSET_TIME_SPAN_COLUMN]),
		LastUpdated: t.readStringSafe(row.Columns[ASSET_LAST_UPDATED_COLUMN]),
		Amount:      t.readStringSafe(row.Columns[ASSET_AMOUNT_COLUMN]),
		Value:       assetValue,
		Locked:      t.readBoolSafe(row.Columns[ASSET_LOCKED_COLUMN]),
		PublicKey:   t.readStringSafe(row.Columns[ASSET_PUBLIC_KEY_COLUMN]),
		Signature:   t.readStringSafe(row.Columns[ASSET_SIGNATURE_COLUMN]),
	}, nil
}

type TradesRecordList struct {
	TradesList     []AnOpenTrade `json:"trades"`
	ProfileCredits string        `json:"credits"`
}

func (t *DNSChaincode) convertToTrades(stub *shim.ChaincodeStub, args []string) (TradesRecordList, error) {
	row, r_err := stub.GetRow(REGISTERED_PROFILE_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: strings.TrimSpace(args[0])}}})
	if r_err != nil || len(row.Columns) == 0 {
		return TradesRecordList{}, fmt.Errorf("Failed to get state for " + args[0])
	}

	qRow, rs_err := stub.GetRows(OPEN_TRADE_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: strings.TrimSpace(t.readStringSafe(row.Columns[PROFILE_PUBLIC_KEY_COLUMN]))}}})
	if rs_err != nil {
		return TradesRecordList{}, fmt.Errorf("Failed to get state for " + args[0])
	}

	var tradeRecords []AnOpenTrade

	for {
		select {
		case currRow, ok := <-qRow:
			if !ok {
				qRow = nil
			} else {
				if len(currRow.Columns) != 0 {
					tradeRecords = append(tradeRecords, AnOpenTrade{
						UserIDA:       t.readStringSafe(currRow.Columns[TRADE_USER_ID_A_COLUMN]),
						PublicKeyA:    t.readStringSafe(currRow.Columns[TRADE_PUBLIC_KEY_A_COLUMN]),
						Status:        t.readStringSafe(currRow.Columns[TRADE_STATUS_COLUMN]),
						UserIDB:       t.readStringSafe(currRow.Columns[TRADE_USER_ID_B_COLUMN]),
						PublicKeyB:    t.readStringSafe(currRow.Columns[TRADE_PUBLIC_KEY_B_COLUMN]),
						CurrentOwner:  t.readStringSafe(currRow.Columns[TRADE_CURRENT_OWNER_COLUMN]),
						AssetName:     t.readStringSafe(currRow.Columns[TRADE_ASSET_NAME_COLUMN]),
						NewAssetName:  t.readStringSafe(currRow.Columns[TRADE_NEW_ASSET_NAME_COLUMN]),
						Amount:        t.readStringSafe(currRow.Columns[TRADE_AMOUNT_COLUMN]),
						StartingValue: t.readStringSafe(currRow.Columns[TRADE_STARTING_VALUE_COLUMN]),
						Value:         t.readStringSafe(currRow.Columns[TRADE_VALUE_COLUMN]),
						TimeSpan:      t.readStringSafe(currRow.Columns[TRADE_TIME_SPAN_COLUMN]),
						Signature:     t.readStringSafe(currRow.Columns[TRADE_SIGNATURE_COLUMN]),
					})
				}
			}
		}
		if qRow == nil {
			break
		}
	}

	return TradesRecordList{TradesList: tradeRecords, ProfileCredits: t.readStringSafe(row.Columns[PROFILE_CREDITS_COLUMN])}, nil
}

func (t *DNSChaincode) convertToProfile(stub *shim.ChaincodeStub, args []string) (Profile, error) {
	if len(args) != 1 {
		return Profile{}, errors.New("Incorrect number of arguments. Expecting 1")
	}

	var JsonBytes []byte
	var row shim.Row
	var err error

	row, err = stub.GetRow(REGISTERED_PROFILE_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: args[0]}}})
	if err != nil || len(row.Columns) == 0 {
		return Profile{}, errors.New("record does not exists")
	}

	var JsonString string
	JsonBytes = t.readBytesSafe(row.Columns[PROFILE_JSON_DATA_COLUMN])
	if JsonBytes != nil {
		JsonString = string(JsonBytes)
	}

	return Profile{
		UserID:    t.readStringSafe(row.Columns[PROFILE_USER_ID_COLUMN]),
		PublicKey: t.readStringSafe(row.Columns[PROFILE_PUBLIC_KEY_COLUMN]),
		JsonData:  JsonString,
		Assets:    t.readStringSafe(row.Columns[PROFILE_OWNED_ASSETS_COLUMN]),
		Credits:   t.readStringSafe(row.Columns[PROFILE_CREDITS_COLUMN]),
	}, nil
}

func (t *DNSChaincode) convertToStats(stub *shim.ChaincodeStub, args []string) (Stats, error) {
	if len(args) != 0 {
		return Stats{}, errors.New("Incorrect number of arguments. Expecting 0")
	}

	var assetState, totalAssetState, profileState, totalRenewAssetsState, totalOpenAssetTradeState, openAssetTradeState, acceptedAssetTradeState, closeAssetTradeState, makeOfferAssetTradeState []byte
	var err error

	assetState, err = stub.GetState(CURRENT_ASSETS_STATS_NAME)
	if err != nil {
		return Stats{}, errors.New("state does not exists")
	}

	profileState, err = stub.GetState(CURRENT_PROFILE_STATS_NAME)
	if err != nil {
		return Stats{}, errors.New("state does not exists")
	}

	totalAssetState, err = stub.GetState(TOTAL_ASSETS_STATS_NAME)
	if err != nil {
		return Stats{}, errors.New("state does not exists")
	}

	totalOpenAssetTradeState, err = stub.GetState(TOTAL_OPEN_ASSET_TRADE_STATS_NAME)
	if err != nil {
		return Stats{}, errors.New("state does not exists")
	}

	totalRenewAssetsState, err = stub.GetState(TOTAL_RENEW_ASSETS_STATS_NAME)
	if err != nil {
		return Stats{}, errors.New("state does not exists")
	}

	openAssetTradeState, err = stub.GetState(OPEN_ASSET_TRADE_STATS_NAME)
	if err != nil {
		return Stats{}, errors.New("state does not exists")
	}

	acceptedAssetTradeState, err = stub.GetState(ACCEPTED_ASSET_TRADE_STATS_NAME)
	if err != nil {
		return Stats{}, errors.New("state does not exists")
	}

	closeAssetTradeState, err = stub.GetState(CLOSE_ASSET_TRADE_STATS_NAME)
	if err != nil {
		return Stats{}, errors.New("state does not exists")
	}

	makeOfferAssetTradeState, err = stub.GetState(MAKE_OFFER_ASSET_TRADE_STATS_NAME)
	if err != nil {
		return Stats{}, errors.New("state does not exists")
	}

	return Stats{
		RegisteredAssets:      string(assetState),
		RegisteredProfiles:    string(profileState),
		TotalRegisteredAssets: string(totalAssetState),
		TotalRenewAssets:      string(totalRenewAssetsState),
		TotalOpenAssetTrades:  string(totalOpenAssetTradeState),
		OpenAssetTrades:       string(openAssetTradeState),
		AcceptAssetTrades:     string(acceptedAssetTradeState),
		CloseAssetTrades:      string(closeAssetTradeState),
		MakeOfferAssetTrades:  string(makeOfferAssetTradeState),
	}, nil
}

func (t *DNSChaincode) readBytesSafe(col *shim.Column) []byte {
	if col == nil {
		return []byte("")
	}

	return col.GetBytes()
}

func (t *DNSChaincode) readStringSafe(col *shim.Column) string {
	if col == nil {
		return ""
	}

	return col.GetString_()
}

func (t *DNSChaincode) readInt32Safe(col *shim.Column) int32 {
	if col == nil {
		return 0
	}

	return col.GetInt32()
}

func (t *DNSChaincode) readBoolSafe(col *shim.Column) bool {
	if col == nil {
		return false
	}

	return col.GetBool()
}

func (t *DNSChaincode) checkAssetOwnership(stub *shim.ChaincodeStub, assetName string, ownerSignature string, ownerPublicKey string, args []string, useKeyFromArgs bool) bool {
	var token string = ""
	for index := 0; index < len(args); index++ {
		if index != 1 {
			token = token + strings.TrimSpace(args[index])
		}
	}

	signByte, _ := hex.DecodeString(strings.Replace(ownerSignature, " ", "", -1))

	var row shim.Row
	var rErr error

	if !useKeyFromArgs {
		row, rErr = stub.GetRow(REGISTERED_ASSETS_TABLE, []shim.Column{{Value: &shim.Column_String_{String_: assetName}}})
		if rErr != nil || len(row.Columns) == 0 {
			return false
		}

		pubByte, _ := hex.DecodeString(strings.Replace(row.Columns[ASSET_PUBLIC_KEY_COLUMN].GetString_(), " ", "", -1))

		key, keyError := t.parsePublicKey(pubByte)
		if keyError == nil {
			return key.Unsign([]byte(token), signByte) == nil
		}
	} else {
		pubByte, _ := hex.DecodeString(strings.Replace(ownerPublicKey, " ", "", -1))
		key, keyError := t.parsePublicKey(pubByte)
		if keyError == nil {
			return key.Unsign([]byte(token), signByte) == nil
		}
	}

	return false
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
