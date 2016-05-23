/*
Copyright 2016 IBM

Licensed under the Apache License, Version 2.0 (the "License")
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Licensed Materials - Property of IBM
© Copyright IBM Corp. 2016
*/
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
    "strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var cpPrefix = "cp:"
var accountPrefix = "acct:"
var accountsKey = "accounts"

var recentLeapYear = 2016

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

func generateCUSIPSuffix(issueDate string, days int) (string, error) {

	t, err := msToTime(issueDate)
	if err != nil {
		return "", err
	}

	suffix := strconv.FormatInt(t.Unix(), 10)
	return suffix, nil

}

const (
	millisPerSecond     = int64(time.Second / time.Millisecond)
	nanosPerMillisecond = int64(time.Millisecond / time.Nanosecond)
)

func msToTime(ms string) (time.Time, error) {
	msInt, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(msInt/millisPerSecond,
		(msInt%millisPerSecond)*nanosPerMillisecond), nil
}



type Owner struct {
	Company string    `json:"company"`
	Quantity int      `json:"quantity"`
}

type CP struct {
	CUSIP     string  `json:"cusip"`
	Ticker    string  `json:"ticker"`
	Par       float64 `json:"par"`
	Qty       int     `json:"qty"`
	Discount  float64 `json:"discount"`
	Maturity  int     `json:"maturity"`
	Owners    []Owner `json:"owner"`
	Issuer    string  `json:"issuer"`
	IssueDate string  `json:"issueDate"`
}

type Account struct {
	ID          string  `json:"id"`
	Prefix      string  `json:"prefix"`
	CashBalance float64 `json:"cashBalance"`
	AssetsIds   []string `json:"assetIds"`
}

type Transaction struct {
	CUSIP       string   `json:"cusip"`
	FromCompany string   `json:"fromCompany"`
	ToCompany   string   `json:"toCompany"`
	Quantity    int      `json:"quantity"`
	Discount    float64  `json:"discount"`
}

func (t *SimpleChaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
    // Initialize the collection of commercial paper keys
    fmt.Println("Initializing paper keys collection")
	var blank []string
	blankBytes, _ := json.Marshal(&blank)
	err := stub.PutState("PaperKeys", blankBytes)
    if err != nil {
        fmt.Println("Failed to initialize paper key collection")
    }

	fmt.Println("Initialization complete")
	return nil, nil
}

func (t *SimpleChaincode) clear(stub *shim.ChaincodeStub, args []string) ([]byte, error)  {
	fmt.Println("Reinitializing paper keys collection for new game")
	var blank []string
	blankBytes, _ := json.Marshal(&blank)
	err = stub.PutState("PaperKeys", blankBytes)
    if err != nil {
        fmt.Println("Failed to reinitialize paper key collection")
    }

	fmt.Println("Reinitialization complete")
	return nil, nil
}

func (t *SimpleChaincode) addOne(stub *shim.ChaincodeStub) ([]byte, error) {
	fmt.Println("trying to add one")
	useracc, err := GetCompany("randy", stub)
	if err != nil {
		fmt.Println("Error from getCompany")
		return nil, err
	}
	useracc.CashBalance = useracc.CashBalance + 1 
	fmt.Println("Marshalling account bytes to write")
	accountBytesToWrite, err := json.Marshal(useracc)
	if err != nil {
		fmt.Println("Error marshalling account")
		return nil, errors.New("Error marshalling account")
	}
	err = stub.PutState(accountPrefix + "randy", accountBytesToWrite)
	if err != nil {
		fmt.Println("Error putting state on accountBytesToWrite")
		return nil, errors.New("Error issuing payouts")
	}
	return nil, nil
}

func (t *SimpleChaincode) createAccounts(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

	//  				0
	// "number of accounts to create"
	var err error
	numAccounts, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Println("error creating accounts with input")
		return nil, errors.New("createAccounts accepts a single integer argument")
	}
	//create a bunch of accounts
	var account Account
	counter := 1
	for counter <= numAccounts {
		var prefix string
		suffix := "000A"
		if counter < 10 {
			prefix = strconv.Itoa(counter) + "0" + suffix
		} else {
			prefix = strconv.Itoa(counter) + suffix
		}
		var assetIds []string
		account = Account{ID: "company" + strconv.Itoa(counter), Prefix: prefix, CashBalance: 1000.0, AssetsIds: assetIds}
		accountBytes, err := json.Marshal(&account)
		if err != nil {
			fmt.Println("error creating account" + account.ID)
			return nil, errors.New("Error creating account " + account.ID)
		}
		err = stub.PutState(accountPrefix+account.ID, accountBytes)
		counter++
		fmt.Println("created account" + accountPrefix + account.ID)
	}

	fmt.Println("Accounts created")
	return nil, nil

}

func (t *SimpleChaincode) createAccount(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
    // Obtain the username to associate with the account
    if len(args) != 1 {
        fmt.Println("Error obtaining username")
        return nil, errors.New("createAccount accepts a single username argument")
    }
    username := args[0]
    
    // Build an account object for the user
    var assetIds []string
    suffix := "000A"
    prefix := username + suffix
    var account = Account{ID: username, Prefix: prefix, CashBalance: 1000.0, AssetsIds: assetIds}
    accountBytes, err := json.Marshal(&account)
    if err != nil {
        fmt.Println("error creating account" + account.ID)
        return nil, errors.New("Error creating account " + account.ID)
    }
    
    fmt.Println("Attempting to get state of any existing account for " + account.ID)
    existingBytes, err := stub.GetState(accountPrefix + account.ID)
	if err == nil {
        
        var company Account
        err = json.Unmarshal(existingBytes, &company)
        if err != nil {
            fmt.Println("Error unmarshalling account " + account.ID + "\n--->: " + err.Error())
            
            if strings.Contains(err.Error(), "unexpected end") {
                fmt.Println("No data means existing account found for " + account.ID + ", initializing account.")
                err = stub.PutState(accountPrefix+account.ID, accountBytes)
                
                if err == nil {
                    fmt.Println("created account" + accountPrefix + account.ID)
                    return nil, nil
                } else {
                    fmt.Println("failed to create initialize account for " + account.ID)
                    return nil, errors.New("failed to initialize an account for " + account.ID + " => " + err.Error())
                }
            } else {
                return nil, errors.New("Error unmarshalling existing account " + account.ID)
            }
        } else {
            fmt.Println("Account already exists for " + account.ID + " " + company.ID)
		    return nil, errors.New("Can't reinitialize existing user " + account.ID)
        }
    } else {
        
        fmt.Println("No existing account found for " + account.ID + ", initializing account.")
        err = stub.PutState(accountPrefix+account.ID, accountBytes)
        
        if err == nil {
            fmt.Println("created account" + accountPrefix + account.ID)
            return nil, nil
        } else {
            fmt.Println("failed to create initialize account for " + account.ID)
            return nil, errors.New("failed to initialize an account for " + account.ID + " => " + err.Error())
        }
        
    }
    
    
}

func (t *SimpleChaincode) issueCommercialPaper(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

	/*		0
		json
	  	{
			"ticker":  "string",
			"par": 0.00,
			"qty": 10,
			"discount": 7.5,
			"maturity": 30,
			"owners": [ // This one is not required
				{
					"company": "company1",
					"quantity": 5
				},
				{
					"company": "company3",
					"quantity": 3
				},
				{
					"company": "company4",
					"quantity": 2
				}
			],				
			"issuer":"company2",
			"issueDate":"1456161763790"  (current time in milliseconds as a string)

		}
	*/
	//need one arg
	if len(args) != 1 {
		fmt.Println("error invalid arguments")
		return nil, errors.New("Incorrect number of arguments. Expecting commercial paper record")
	}

	var cp CP
	var err error
	var account Account

	fmt.Println("Unmarshalling CP")
	err = json.Unmarshal([]byte(args[0]), &cp)
	if err != nil {
		fmt.Println("error invalid paper issue")
		return nil, errors.New("Invalid commercial paper issue")
	}

	//generate the CUSIP
	//get account prefix
	fmt.Println("Getting state of - " + accountPrefix + cp.Issuer)
	accountBytes, err := stub.GetState(accountPrefix + cp.Issuer)
	if err != nil {
		fmt.Println("Error Getting state of - " + accountPrefix + cp.Issuer)
		return nil, errors.New("Error retrieving account " + cp.Issuer)
	}
	err = json.Unmarshal(accountBytes, &account)
	if err != nil {
		fmt.Println("Error Unmarshalling accountBytes")
		return nil, errors.New("Error retrieving account " + cp.Issuer)
	}
	
	account.AssetsIds = append(account.AssetsIds, cp.CUSIP)

	// Set the issuer to be the owner of all quantity
	var owner Owner
	owner.Company = cp.Issuer
	owner.Quantity = cp.Qty
	
	cp.Owners = append(cp.Owners, owner)

	suffix, err := generateCUSIPSuffix(cp.IssueDate, cp.Maturity)
	if err != nil {
		fmt.Println("Error generating cusip")
		return nil, errors.New("Error generating CUSIP")
	}

	fmt.Println("Marshalling CP bytes")
	cp.CUSIP = account.Prefix + suffix
	
	fmt.Println("Getting State on CP " + cp.CUSIP)
	cpRxBytes, err := stub.GetState(cpPrefix+cp.CUSIP)
	if cpRxBytes == nil {
		fmt.Println("CUSIP does not exist, creating it")
		cpBytes, err := json.Marshal(&cp)
		if err != nil {
			fmt.Println("Error marshalling cp")
			return nil, errors.New("Error issuing commercial paper")
		}
		err = stub.PutState(cpPrefix+cp.CUSIP, cpBytes)
		if err != nil {
			fmt.Println("Error issuing paper")
			return nil, errors.New("Error issuing commercial paper")
		}

		fmt.Println("Marshalling account bytes to write")
		accountBytesToWrite, err := json.Marshal(&account)
		if err != nil {
			fmt.Println("Error marshalling account")
			return nil, errors.New("Error issuing commercial paper")
		}
		err = stub.PutState(accountPrefix + cp.Issuer, accountBytesToWrite)
		if err != nil {
			fmt.Println("Error putting state on accountBytesToWrite")
			return nil, errors.New("Error issuing commercial paper")
		}
		
		
		// Update the paper keys by adding the new key
		fmt.Println("Getting Paper Keys")
		keysBytes, err := stub.GetState("PaperKeys")
		if err != nil {
			fmt.Println("Error retrieving paper keys")
			return nil, errors.New("Error retrieving paper keys")
		}
		var keys []string
		err = json.Unmarshal(keysBytes, &keys)
		if err != nil {
			fmt.Println("Error unmarshel keys")
			return nil, errors.New("Error unmarshalling paper keys ")
		}
		
		fmt.Println("Appending the new key to Paper Keys")
		foundKey := false
		for _, key := range keys {
			if key == cpPrefix+cp.CUSIP {
				foundKey = true
			}
		}
		if foundKey == false {
			keys = append(keys, cpPrefix+cp.CUSIP)
			keysBytesToWrite, err := json.Marshal(&keys)
			if err != nil {
				fmt.Println("Error marshalling keys")
				return nil, errors.New("Error marshalling the keys")
			}
			fmt.Println("Put state on PaperKeys")
			err = stub.PutState("PaperKeys", keysBytesToWrite)
			if err != nil {
				fmt.Println("Error writting keys back")
				return nil, errors.New("Error writing the keys back")
			}
		}
		
		fmt.Println("Issue commercial paper %+v\n", cp)
		return nil, nil
	} else {
		fmt.Println("CUSIP exists")
		
		var cprx CP
		fmt.Println("Unmarshalling CP " + cp.CUSIP)
		err = json.Unmarshal(cpRxBytes, &cprx)
		if err != nil {
			fmt.Println("Error unmarshalling cp " + cp.CUSIP)
			return nil, errors.New("Error unmarshalling cp " + cp.CUSIP)
		}
		
		cprx.Qty = cprx.Qty + cp.Qty
		
		for key, val := range cprx.Owners {
			if val.Company == cp.Issuer {
				cprx.Owners[key].Quantity += cp.Qty
				break
			}
		}
				
		cpWriteBytes, err := json.Marshal(&cprx)
		if err != nil {
			fmt.Println("Error marshalling cp")
			return nil, errors.New("Error issuing commercial paper")
		}
		err = stub.PutState(cpPrefix+cp.CUSIP, cpWriteBytes)
		if err != nil {
			fmt.Println("Error issuing paper")
			return nil, errors.New("Error issuing commercial paper")
		}

		fmt.Println("Updated commercial paper %+v\n", cprx)
		return nil, nil
	}
}


func GetAllCPs(stub *shim.ChaincodeStub) ([]CP, error){
	
	var allCPs []CP
	
	// Get list of all the keys
	keysBytes, err := stub.GetState("PaperKeys")
	if err != nil {
		fmt.Println("Error retrieving paper keys")
		return nil, errors.New("Error retrieving paper keys")
	}
	var keys []string
	err = json.Unmarshal(keysBytes, &keys)
	if err != nil {
		fmt.Println("GetAllCPs: Error unmarshalling paper keys print")
		return nil, errors.New("GetAllCPs: Error unmarshalling paper keys")
	}

	// Get all the cps
	for _, value := range keys {
		cpBytes, err := stub.GetState(value)
		
		var cp CP
		err = json.Unmarshal(cpBytes, &cp)
		if err != nil {
			fmt.Println("Error retrieving cp " + value)
			return nil, errors.New("Error retrieving cp " + value)
		}
		
		fmt.Println("Appending CP" + value)
		allCPs = append(allCPs, cp)
	}	
	
	return allCPs, nil
}

func GetCP(cpid string, stub *shim.ChaincodeStub) (CP, error){
	var cp CP

	cpBytes, err := stub.GetState(cpid)
	if err != nil {
		fmt.Println("Error retrieving cp " + cpid)
		return cp, errors.New("Error retrieving cp " + cpid)
	}
		
	err = json.Unmarshal(cpBytes, &cp)
	if err != nil {
		fmt.Println("Error unmarshalling cp " + cpid)
		return cp, errors.New("Error unmarshalling cp " + cpid)
	}
		
	return cp, nil
}


func GetCompany(companyID string, stub *shim.ChaincodeStub) (Account, error){
	var company Account
	companyBytes, err := stub.GetState(accountPrefix+companyID)
	if err != nil {
		fmt.Println("Account not found " + companyID)
		return company, errors.New("Account not found " + companyID)
	}

	err = json.Unmarshal(companyBytes, &company)
	if err != nil {
		fmt.Println("Error unmarshalling account " + companyID + "\n err:" + err.Error())
		return company, errors.New("Error unmarshalling account " + companyID)
	}
	
	return company, nil
}

func (t *SimpleChaincode) transferPaper(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	/*- Function to announce winner and pay out bets
	- currently defaults to paying out x2 the original wager, can make this dynamic with an argument
	grabs all bets/cps by using the PaperKeys array
		- will need to implement some kind of time tracking feature for last game, might create a block with a unique CUSIP 
	- compares qty (currently our variable for player) if it matches we credit account with x2 the wager, otherwise we deduct their wager
	- we clear the paperkeys array and put it back in prep for the next game
	- get account use username000Ausername
	- Bet/CP stores username as Issuer
	*/ 
	//need one arg
	if len(args) != 1 {
		fmt.Println("error invalid arguments")
		return nil, errors.New("Incorrect number of arguments.")
	}
	fmt.Println("DEBUG: okay number of arguments")
	var err error
	fmt.Println("DEBUG: getting list of keys in byte form")
	// Get list of all the keys
	keysBytes, err := stub.GetState("PaperKeys")
	if err != nil {
		fmt.Println("Error retrieving paper keys")
		return nil, errors.New("Error retrieving paper keys")
	}
	fmt.Println("DEBUG: unmarshalling keys")
	var keys []string
	err = json.Unmarshal(keysBytes, &keys)
	if err != nil {
		fmt.Println("Error payout failed unmarshalling paper keys")
		return nil, errors.New("payout failed to unmarshal keys")
	}
	winner, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Println("error parsing winner")
		return nil, errors.New("error parsing winner")
	}
	fmt.Println("DEBUG: Winner declared, Paying out bets.")
	fmt.Println("DEBUG: calling GetAllCPs")
	
	var allCPs []CP
	
	// Get all the cps
	for _, value := range keys {
		fmt.Println("DEBUG: getting CP")
		cpBytes, err := stub.GetState(value)
		
		var cp CP
		err = json.Unmarshal(cpBytes, &cp)
		if err != nil {
			fmt.Println("Error retrieving cp " + value)
			return nil, errors.New("Error retrieving cp " + value)
		}
		
		fmt.Println("Appending CP" + value)
		allCPs = append(allCPs, cp)
	}
	fmt.Println("DEBUG: AllCPs retrieved")

	// Get all the cps
	for _, value := range allCPs {
		fmt.Println("DEBUG: getting account for " + value.Issuer)
		useracc, err := GetCompany(value.Issuer, stub)
		if err != nil {
			fmt.Println("Error from getCompany")
			return nil, err
		}
		if value.Qty == winner {
			useracc.CashBalance = useracc.CashBalance + (2 * value.Par)
		}
		if value.Qty != winner {
			useracc.CashBalance = useracc.CashBalance - value.Par
		}
		fmt.Println("Marshalling account bytes to write")
		accountBytesToWrite, err := json.Marshal(useracc)
		if err != nil {
			fmt.Println("Error marshalling account")
			return nil, errors.New("Error marshalling account")
		}
		err = stub.PutState(accountPrefix + value.Issuer, accountBytesToWrite)
		if err != nil {
			fmt.Println("Error putting state on accountBytesToWrite")
			return nil, errors.New("Error issuing payouts")
		}
	}
    fmt.Println("Reinitializing paper keys collection for new game")
	var blank []string
	blankBytes, _ := json.Marshal(&blank)
	err = stub.PutState("PaperKeys", blankBytes)
    if err != nil {
        fmt.Println("Failed to reinitialize paper key collection")
    }

	fmt.Println("Reinitialization complete")
	return nil, nil
}
func (t *SimpleChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	//need one arg
	if len(args) < 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting ......")
	}

	if args[0] == "GetAllCPs" {
		fmt.Println("Getting all CPs")
		allCPs, err := GetAllCPs(stub)
		if err != nil {
			fmt.Println("Error from getallcps")
			return nil, err
		} else {
			allCPsBytes, err1 := json.Marshal(&allCPs)
			if err1 != nil {
				fmt.Println("Error marshalling allcps")
				return nil, err1
			}	
			fmt.Println("All success, returning allcps")
			return allCPsBytes, nil		 
		}
	} else if args[0] == "GetCP" {
		fmt.Println("Getting particular cp")
		cp, err := GetCP(args[1], stub)
		if err != nil {
			fmt.Println("Error Getting particular cp")
			return nil, err
		} else {
			cpBytes, err1 := json.Marshal(&cp)
			if err1 != nil {
				fmt.Println("Error marshalling the cp")
				return nil, err1
			}	
			fmt.Println("All success, returning the cp - test")
			return cpBytes, nil		 
		}
	} else if args[0] == "GetCompany" {
		fmt.Println("Getting the company")
		company, err := GetCompany(args[1], stub)
		if err != nil {
			fmt.Println("Error from getCompany")
			return nil, err
		} else {
			companyBytes, err1 := json.Marshal(&company)
			if err1 != nil {
				fmt.Println("Error marshalling the company")
				return nil, err1
			}	
			fmt.Println("All success, returning the company")
			return companyBytes, nil		 
		}
	} else {
		fmt.Println("Generic Query call")
		bytes, err := stub.GetState(args[0])

		if err != nil {
			fmt.Println("Some error happenend")
			return nil, errors.New("Some Error happened")
		}

		fmt.Println("All success, returning from generic")
		return bytes, nil		
	}
}

func (t *SimpleChaincode) Run(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("run is running " + function)
	return t.Invoke(stub, function, args)
}

func (t *SimpleChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)
	
	if function == "issueCommercialPaper" {
		fmt.Println("Firing issueCommercialPaper")
		//Create an asset with some value
		return t.issueCommercialPaper(stub, args)
	} else if function == "transferPaper" {
		fmt.Println("Firing cretransferPaperateAccounts")
		return t.transferPaper(stub, args)
	} else if function == "createAccounts" {
		fmt.Println("Firing createAccounts")
		return t.createAccounts(stub, args)
	} else if function == "createAccount" {
        fmt.Println("Firing createAccount")
        return t.createAccount(stub, args)
    } else if function == "init" {
        fmt.Println("Firing init")
        return t.Init(stub, "init", args)
	} else if function == "clear" {
		fmt.Println("Firing clear")
		return t.clear(stub)
	}

	return nil, errors.New("Received unknown function invocation")
}

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Println("Error starting Simple chaincode: %s", err)
	}
}

//lookup tables for last two digits of CUSIP
var seventhDigit = map[int]string{
	1:  "A",
	2:  "B",
	3:  "C",
	4:  "D",
	5:  "E",
	6:  "F",
	7:  "G",
	8:  "H",
	9:  "J",
	10: "K",
	11: "L",
	12: "M",
	13: "N",
	14: "P",
	15: "Q",
	16: "R",
	17: "S",
	18: "T",
	19: "U",
	20: "V",
	21: "W",
	22: "X",
	23: "Y",
	24: "Z",
}

var eigthDigit = map[int]string{
	1:  "1",
	2:  "2",
	3:  "3",
	4:  "4",
	5:  "5",
	6:  "6",
	7:  "7",
	8:  "8",
	9:  "9",
	10: "A",
	11: "B",
	12: "C",
	13: "D",
	14: "E",
	15: "F",
	16: "G",
	17: "H",
	18: "J",
	19: "K",
	20: "L",
	21: "M",
	22: "N",
	23: "P",
	24: "Q",
	25: "R",
	26: "S",
	27: "T",
	28: "U",
	29: "V",
	30: "W",
	31: "X",
}
