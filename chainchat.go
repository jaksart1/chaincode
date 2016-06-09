package main

import (
    "errors"
	"fmt"
	"strconv"
	"encoding/json"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

type SimpleChaincode struct {
}

var chatIndexStr = "_chatindex"
var openTradesStr = "_opentrades"

type Message struct{ 
    Msg string `json:"msg"`
    To string `json:"to"`
    User string `json:"user"`
}

//Main function
func main() {
    
    err := shim.Start(new(SimpleChaincode))
    if err != nil {
        fmt.Printf("Error starting Simple chaincode: %s", err)
    }
}
//Init funciton
func (t *SimpleChaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	var Aval int
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	// Initialize the chaincode
	Aval, err = strconv.Atoi(args[0])
	if err != nil {
		return nil, errors.New("Expecting integer value for asset holding")
	}

	// Write the state to the ledger
	err = stub.PutState("abc", []byte(strconv.Itoa(Aval)))				//making a test var "abc", I find it handy to read/write to it right away to test the network
	if err != nil {
		return nil, err
	}
	
	var empty []string
	jsonAsBytes, _ := json.Marshal(empty)								//marshal an emtpy array of strings to clear the index
	err = stub.PutState(chatIndexStr, jsonAsBytes)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

//Run
func (t *SimpleChaincode) Run(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("run is running " + function)
	return t.Invoke(stub, function, args)
}

//Invoke
func (t *SimpleChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "init" {													//initialize the chaincode state, used as reset
		return t.Init(stub, "init", args)
	} else if function == "delete" {										//deletes an entity from its state
		return t.Delete(stub, args)
	} else if function == "write" {											//writes a value to the chaincode state
		return t.Write(stub, args)
	} else if function == "init_msg" {									//create a new message
		return t.init_msg(stub, args)
	} 
	fmt.Println("invoke did not find func: " + function)					//error

	return nil, errors.New("Received unknown function invocation")
}

//Query
func (t *SimpleChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

	// Handle different functions
	if function == "read" {													//read a variable
		return t.read(stub, args)
	}
	fmt.Println("query did not find func: " + function)						//error

	return nil, errors.New("Received unknown function query")
}

//Read
func (t *SimpleChaincode) read(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	var name, jsonResp string
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting name of the var to query")
	}

	name = args[0]
	valAsbytes, err := stub.GetState(name)									//get the var from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + name + "\"}"
		return nil, errors.New(jsonResp)
	}

	return valAsbytes, nil													//send it onward
}


//Delete
func (t *SimpleChaincode) Delete(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}
	
	name := args[0]
	err := stub.DelState(name)													//remove the key from chaincode state
	if err != nil {
		return nil, errors.New("Failed to delete state")
	}

	//get the marble index
	msgAsBytes, err := stub.GetState(chatIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get chat index")
	}
	var chatIndex []string
	json.Unmarshal(msgAsBytes, &chatIndex)								//un stringify it aka JSON.parse()
	
	//remove marble from index
	for i,val := range chatIndex{
		fmt.Println(strconv.Itoa(i) + " - looking at " + val + " for " + name)
		if val == name{															//find the correct marble
			fmt.Println("found marble")
			chatIndex = append(chatIndex[:i], chatIndex[i+1:]...)			//remove it
			for x:= range chatIndex{											//debug prints...
				fmt.Println(string(x) + " - " + chatIndex[x])
			}
			break
		}
	}
	jsonAsBytes, _ := json.Marshal(chatIndex)									//save new index
	err = stub.PutState(chatIndexStr, jsonAsBytes)
	return nil, nil
}

//Write
func (t *SimpleChaincode) Write(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	var name, value string // Entities
	var err error
	fmt.Println("running write()")

	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2. name of the variable and value to set")
	}

	name = args[0]															//rename for funsies
	value = args[1]
	err = stub.PutState(name, []byte(value))								//write the variable into the chaincode state
	if err != nil {
		return nil, err
	}
	return nil, nil
}

//Init message
func (t *SimpleChaincode) init_msg(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	var err error

	//   0       1       2    
	// "asdf", "You",   "Me"
	if len(args) != 3 {
		return nil, errors.New("Incorrect number of arguments. Expecting 4")
	}

	fmt.Println("- start init message")
	if len(args[0]) <= 0 {
		return nil, errors.New("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return nil, errors.New("3rd argument must be a non-empty string")
	}


	to := strings.ToLower(args[1])
	user := strings.ToLower(args[2])

	str := `{"msg": "` + args[0] + `", "To": "` + to + `, "user": "` + user + `"}`
	err = stub.PutState(args[0], []byte(str))								//store message with id as key
	if err != nil {
		return nil, err
	}
		
	//get the chat index
	msgAsBytes, err := stub.GetState(chatIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get marble index")
	}
	var chatIndex []string
	json.Unmarshal(msgAsBytes, &chatIndex)							//un stringify it aka JSON.parse()
	
	//append
	chatIndex = append(chatIndex, args[0])								//add marble name to index list
	fmt.Println("! chat index: ", chatIndex)
	jsonAsBytes, _ := json.Marshal(chatIndex)
	err = stub.PutState(chatIndexStr, jsonAsBytes)						//store name of marble

	fmt.Println("- end init marble")
	return nil, nil
}
