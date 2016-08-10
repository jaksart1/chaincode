/*
Copyright IBM Corp 2016 All Rights Reserved.

Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/

package main

import (
	"encoding/json"
	"strconv"
	"errors"
	"fmt"
	"bytes"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// ChainchatChaincode implementation
type ChainchatChaincode struct {
}

// MessageResults stores an array of retrieved messages
type MessageResults struct {
	Messages	[]Message	`json:"messages"`
}

// Message stores a message and its related info
type Message struct {
	ID			uint64	`json:"id"`
	ReceiverID	string 	`json:"recvID"`
	SenderID	string 	`json:"senderID"`
	ToRoom		bool	`json:"toRoom"`
	Message		string 	`json:"message"`
	Timestamp	string 	`json:"timestamp"`
}

// Stores the index of each column in the messages table
const (
	MsgCatchAllCol = iota
	MsgReceiverIDCol
	MsgIDCol
	MsgCol
)

// User stores user information and its related info
type User struct {
	ID		string	`json:"id"`
	Name	string	`json:"name"`
	PubKey	string	`json:"pubKey"`
	Room	string	`json:"room"`
	Active	bool	`json:"active"`
}

// Stores the index of each column in the users table
const (
	UserCatchAllCol = iota
	UserIDCol
	UserCol
)

// Room stores the name and ID of a room
type Room struct {
	ID			string		`json:"id"`
	Name		string		`json:"name"`
	UsersIn		[]string	`json:"usersIn"`
	CreatedBy	string		`json:"createdBy"`
	CreatedOn	string		`json:"createdOn"`
}

// Stores the index of each column in the rooms table
const (
	RoomCatchAllCol = iota
	RoomIDCol
	RoomCol
)

// Main initializes the shim
func main() {
	err := shim.Start(new(ChainchatChaincode))
	if err != nil {
		fmt.Printf("Error starting Chainchat chaincode: %s", err)
	}
}

// Init prepares the ledger for the chat application
func (t *ChainchatChaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	if function == "init" {
		if len(args) < 1 {
			fmt.Println("[ERROR] Not enough arguments to initialize chaincode, need at least 1")
			return nil, errors.New("Unexpected number of arguments. Need 1 to create the initial room")
		}

		// Create the table where all user structs will be stored
		userTableErr := stub.CreateTable("users", []*shim.ColumnDefinition{
			&shim.ColumnDefinition{Name: "CatchAll", Type: shim.ColumnDefinition_STRING, Key: true},
			&shim.ColumnDefinition{Name: "ID", Type: shim.ColumnDefinition_STRING, Key: true},
			&shim.ColumnDefinition{Name: "User", Type: shim.ColumnDefinition_BYTES, Key: false},
		})

		// Handle table creation errors
		if userTableErr != nil {
			fmt.Println(fmt.Sprintf("[ERROR] Could not create users table: %s", userTableErr.Error()))
			return nil, userTableErr
		}

		// Create the table where all room structs will be stored
		roomTableErr := stub.CreateTable("rooms", []*shim.ColumnDefinition{
			&shim.ColumnDefinition{Name: "CatchAll", Type: shim.ColumnDefinition_STRING, Key: true},
			&shim.ColumnDefinition{Name: "ID", Type: shim.ColumnDefinition_STRING, Key: true},
			&shim.ColumnDefinition{Name: "Room", Type: shim.ColumnDefinition_BYTES, Key: false},
		})

		// Handle table creation errors
		if roomTableErr != nil {
			fmt.Println(fmt.Sprintf("[ERROR] Could not create rooms table: %s", roomTableErr.Error()))
			return nil, roomTableErr
		}

		// Create the table where all messages will be stored
		messageTableErr := stub.CreateTable("messages", []*shim.ColumnDefinition{
			&shim.ColumnDefinition{Name: "CatchAll", Type: shim.ColumnDefinition_STRING, Key: true},
			&shim.ColumnDefinition{Name: "ReceiverID", Type: shim.ColumnDefinition_STRING, Key: true},
			&shim.ColumnDefinition{Name: "ID", Type: shim.ColumnDefinition_UINT64, Key: true},
			&shim.ColumnDefinition{Name: "Message", Type: shim.ColumnDefinition_BYTES, Key: false},
		})

		// Handle table creation errors
		if messageTableErr != nil {
			fmt.Println(fmt.Sprintf("[ERROR] Could not create messages table: %s", messageTableErr.Error()))
			return nil, messageTableErr
		}

		// Initialize the message counter to 0
		counterErr := stub.PutState("counter", []byte("0"))

		// Handle counter state initialization errors
		if counterErr != nil {
			fmt.Println(fmt.Sprintf("[ERROR] Could not initialize message counter to 0: %s", counterErr))
			return nil, counterErr
		}
	}

	return nil, nil
}

// Invoke handles all chat functions which are saved to the blockchain
func (t *ChainchatChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	if function == "init" {
		// Initialize the chaincode state, used as reset
		return t.Init(stub, function, args)
	} else if function == "writeMsg" {
		// Write the message to the ledger for eventual retrieval by the receiver
		return nil, t.WriteMessage(stub, args)
	} else if function == "deleteMsgs" {
		// Deletes the specified messages received by the specified user
		return nil, t.DeleteMessages(stub, args)
	} else if function == "createRoom" {
		// Creates a new room
		return nil, t.CreateRoom(stub, args)
	} else if function == "deleteRoom" {
		// Removes all users from the room and deletes it
		return nil, t.SafelyDeleteRoom(stub, args)
	} else if function == "removeUserFromRoom" {
		// Adds the given user to the specified room
		return nil, t.SafelyRemoveUserFromRoom(stub, args)
	} else if function == "addUserToRoom" {
		// Removes the given user from the specified room
		return nil, t.SafelyAddUserToRoom(stub, args)
	} else if function == "addUser" {
		// Adds a user to the ledger
		return nil, t.AddUser(stub, args)
	} else if function == "deleteUser" {
		// Deletes a user from the ledger
		return nil, t.DeleteUser(stub, args)
	}

	// The requested invoke function was not recognized
	errStr := fmt.Sprintf("[ERROR] Did not recognize invocation: %s", function)
	fmt.Println(errStr)
	return nil, errors.New(errStr)
}

// Takes the 4 components of a message from the args array and writes it to the ledger
// args[0] = Receiver's user ID
// args[1] = Sender's user ID
// args[2] = "true" if the receiver ID a room's ID, anything else if the ID is a user ID
// args[3] = The message contents
// args[4] = A UNIX-style UTC timestamp
func (t *ChainchatChaincode) WriteMessage(stub *shim.ChaincodeStub, args []string) error {
	// Check we have the requisite info
	if len(args) < 4 {
		errStr := fmt.Sprintf("[ERROR] Not enough arguments to write a message, expected 4 got %d", len(args))
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	// Build the message struct out
	id, idErr := t.getAndIncrementMsgCounter(stub)
	if idErr != nil {
		errStr := fmt.Sprintf("[ERROR] Could not get and increment the message counter: %s", idErr.Error())
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	recvID := args[0]
	sendID := args[1]
	toRoom := strings.Compare(args[2], "true") == 0
	msgContent := args[3]
	timestamp := args[4]

	msg := Message{
		ID: id,
		ReceiverID: recvID,
		SenderID: sendID,
		ToRoom: toRoom,
		Message: msgContent,
		Timestamp: timestamp,
	}

	// Prepare the message struct for storage by marshalling it into bytes
	msgBytes, marshalErr := json.Marshal(msg)
	if marshalErr != nil {
		errStr := fmt.Sprintf("[ERROR] Could not marshal message into bytes: %s", marshalErr.Error())
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	// Write the message to the ledger
	rowAdded, rowErr := stub.InsertRow("messages", shim.Row{
		Columns: []*shim.Column{
			{&shim.Column_String_{String_: "message"}},
			{&shim.Column_String_{String_: recvID}},
			{&shim.Column_Uint64{Uint64: id}},
			{&shim.Column_Bytes{Bytes: msgBytes}},
		},
	})

	// Handle error adding the message to the ledger
	if rowErr != nil || !rowAdded {
		errStr := fmt.Sprintf("[ERROR] Could not add message to the ledger: %s", rowErr.Error())
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	return nil
}

// DeleteMessages deletes all messages sent to a particular user with the given IDs
// args[0] = Receiver ID
// args[1..] = The ID of each message to delete
func (t *ChainchatChaincode) DeleteMessages(stub *shim.ChaincodeStub, args []string) error {
	// Make sure we have enough info to delete some messages
	if len(args) < 2 {
		errStr := fmt.Sprintf("[ERROR] Need at least 2 arguments to delete messages, received %d", len(args))
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	// Retrieve the receiver ID of the messages to delete
	recvID := args[0]

	// Iterate through each message ID deleting each one
	for i := 1; i < len(args); i++ {
		// Get the message ID as a string from the args array and convert it to a uint64
		msgIDStr := args[i]
		msgID, convErr := strconv.ParseUint(msgIDStr, 10, 64)

		// If conversion succeeded delete the message, otherwise indicate failure for that message
		if convErr != nil {
			fmt.Printf(fmt.Sprintf("[ERROR] Could not convert message ID %s to a number for deletion", msgIDStr))
		} else {
			delErr := stub.DeleteRow("messages", []shim.Column{
				shim.Column{Value: &shim.Column_String_{String_: "message"}},
				shim.Column{Value: &shim.Column_String_{String_: recvID}},
				shim.Column{Value: &shim.Column_Uint64{Uint64: msgID}},
			})

			if delErr != nil {
				fmt.Printf("[ERROR] Error deleting row: %s", delErr)
			}
		}
	}

	return nil
}

// CreateRoom adds a new room to the ledger using the given info
// args[0] = Unique ID of the room
// args[1] = Display name of the room
// args[2] = User ID of the room's creator
// args[3] = UNIX-style UTC timestamp when the room was created
func (t *ChainchatChaincode) CreateRoom(stub *shim.ChaincodeStub, args []string) error {
	// Make sure we have enough info to create a room
	if len(args) < 4 {
		errStr := fmt.Sprintf("[ERROR] Expected 4 arguments to create a room, received %d", len(args))
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	// Extract room info from the arguments rarray
	roomID := args[0]
	roomName := args[1]
	creator := args[2]
	timestamp := args[3]

	// Create the room struct
	newRoom := Room{
		ID: roomID,
		Name: roomName,
		UsersIn: []string{},
		CreatedBy: creator,
		CreatedOn: timestamp,
	}

	// Marshal the struct into bytes for storage to the ledger
	roomBytes, marshalErr := json.Marshal(newRoom)
	if marshalErr != nil {
		errStr := fmt.Sprintf("[ERROR] Could not marshal the room into bytes for storage: %s", marshalErr.Error())
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	// Add the new room to the ledger
	rowAdded, rowErr := stub.InsertRow("rooms", shim.Row{
		Columns: []*shim.Column{
			{&shim.Column_String_{String_: "room"}},
			{&shim.Column_String_{String_: roomID}},
			{&shim.Column_Bytes{Bytes: roomBytes}},
		},
	})

	// Handle error adding the message to the ledger
	if rowErr != nil || !rowAdded {
		errStr := fmt.Sprintf("[ERROR] Could not add room to the ledger: %s", rowErr.Error())
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	return nil
}

// SafelyDeleteRoom disassociates all users from the given room and then deletes the room from the ledger
// args[0] = The room ID to delete
func (t *ChainchatChaincode) SafelyDeleteRoom(stub *shim.ChaincodeStub, args []string) error {
	// Make sure we have the ID of the room to delete
	if len(args) < 1 {
		errStr := fmt.Sprintf("[ERROR] Expected 1 argument to delete a room, received %d", len(args))
		fmt.Printf(errStr)
		return errors.New(errStr)
	}

	// Extract the room to delete from the arguments
	roomID := args[0]

	// Get the room from the ledger
	room, roomErr := t.getRoom(stub, roomID)
	if roomErr != nil {
		errStr := fmt.Sprintf("[ERROR] Could not get room: %s", roomErr.Error())
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	// Update the users in the room and catch problem users along the way
	var errorArr []error
	for i := 0; i < len(room.UsersIn); i++ {
		// Extract the user ID
		userID := room.UsersIn[i]

		// Update the given user's room to be blank
		updateErr := t.updateUserRoom(stub, userID, "")
		if updateErr != nil {
			errStr := fmt.Sprintf("Could not update user's room: %s", updateErr.Error())
			errorArr = append(errorArr, errors.New(errStr))
		}
	}

	// If there were errors deleting users from their room, do not go through removing the room from the ledger
	if len(errorArr) > 0 {
		var updateUsersErrs bytes.Buffer
		updateUsersErrs.WriteString("[ERROR] Encountered the following issues while removing users from their room:\n")
		for j := 0; j < len(errorArr); j++ {
			currErr := errorArr[j]
			updateUsersErrs.WriteString(fmt.Sprintf("\t%s\n", currErr.Error()))
		}

		fmt.Println(updateUsersErrs.String())
		return errors.New(updateUsersErrs.String())
	}

	// Assuming no errors, let's now delete the room from the ledger
	deleteErr := t.deleteRoom(stub, roomID)
	if deleteErr != nil {
		errStr := fmt.Sprintf("[ERROR] Could not delete room %s: %s", roomID, deleteErr.Error())
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	return nil
}

// Safely removes a user from a room by updating both the user's record and the room's record
// args[0] = The user ID who is going to be removed from the room
func (t *ChainchatChaincode) SafelyRemoveUserFromRoom(stub *shim.ChaincodeStub, args []string) error {
	// Error if we don't have the right info
	if len(args) < 1 {
		errStr := fmt.Sprintf("[ERROR] Expected at least 1 argument to remove user from a room, received %d", len(args))
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	// Extract info from arguments array
	userID := args[0]

	// Get the user from the ledger
	user, userErr := t.getUser(stub, userID)
	if userErr != nil {
		errStr := fmt.Sprintf("[ERROR] Unable to retrieve user %s: %s", userID, userErr.Error())
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	// Extract the user's current room and remove the user from that room
	roomID := user.Room
	removeFromRoomErr := t.removeUserFromRoom(stub, userID, roomID)
	if removeFromRoomErr != nil {
		errStr := fmt.Sprintf("[ERROR] Could not remove a user from the room record: %s", removeFromRoomErr.Error())
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	// Update the room the user is currently in, setting it to be blank
	updateUserErr := t.updateUserRoom(stub, userID, "")
	if updateUserErr != nil {
		errStr := fmt.Sprintf("[ERROR] Could not update user's room record: %s", updateUserErr.Error())
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	return nil
}

// Safely adds a user to a room by updating both the user's record and the room's record
// args[0] = The user ID who is going to be added to a room
// args[1] = The room to add the user to
func (t *ChainchatChaincode) SafelyAddUserToRoom(stub *shim.ChaincodeStub, args []string) error {
	// Check correct number of arguments
	if len(args) < 2 {
		errStr := fmt.Sprintf("[ERROR] Expected 2 arguments to add a user to a room, received %d", len(args))
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	// Extract info from args array
	userID := args[0]
	roomID := args[1]

	updateUserErr := t.updateUserRoom(stub, userID, roomID)
	if updateUserErr != nil {
		errStr := fmt.Sprintf("[ERROR] Unable to change room of user %s: %s", userID, updateUserErr.Error())
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	// Update the room
	roomUpdateErr := t.addUserToRoom(stub, userID, roomID)
	if roomUpdateErr != nil {
		errStr := fmt.Sprintf("[ERROR] Unable to add user %s to room %s: %s", userID, roomID, roomUpdateErr.Error())
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	return nil
}

// Adds a new user to the ledger
// args[0] = User's ID
// args[1] = User's name
// args[2] = User's public key
// args[3] = User's room
// args[4] = Empty string if the user is logged out, any string if the user is logged in
func (t *ChainchatChaincode) AddUser(stub *shim.ChaincodeStub, args []string) error {
	// Make sure we have the right number of arguments
	if len(args) < 5 {
		errStr := fmt.Sprintf("[ERROR] Expected 5 arguments to create a new user, received %d", len(args))
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	// Extract information from arguments array
	userID := args[0]
	userName := args[1]
	userPubKey := args[2]
	userRoom := args[3]
	userActive := len(args[4]) > 0

	// Build the user struct
	user := User{
		ID: userID,
		Name: userName,
		PubKey: userPubKey,
		Room: userRoom,
		Active: userActive,
	}

	// Marshal the user into bytes for storage
	userBytes, marshalErr := json.Marshal(user)
	if marshalErr != nil {
		errStr := fmt.Sprintf("[ERROR] Could not marshal new user %s into bytes for storage: %s", userID, marshalErr.Error())
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	// Add the user to the user's table
	rowAdded, rowErr := stub.InsertRow("users", shim.Row{
		Columns: []*shim.Column{
			{&shim.Column_String_{String_: "user"}},
			{&shim.Column_String_{String_: userID}},
			{&shim.Column_Bytes{Bytes: userBytes}},
		},
	})

	// Deal with ledger errors
	if rowErr != nil {
		errStr := fmt.Sprintf("[ERROR] Could not add user %s to user's table: %s", userID, rowErr.Error())
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	// Deal with user already exists error
	if !rowAdded {
		errStr := fmt.Sprintf("[ERROR] A user with ID %s already exists, cannot add to table", userID)
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	// If a room was given, add the user to the rooms table
	if len(userRoom) > 0 {
		roomErr := t.addUserToRoom(stub, userID, userRoom)
		if roomErr != nil {
			errStr := fmt.Sprintf("[ERROR] Could not add new user %s to given room: %s", userID, roomErr.Error())
			fmt.Println(errStr)
			return errors.New(errStr)
		}
	}

	return nil
}

// Removes a user from the ledger and updates the room ledger
// args[0] = The user ID to remove
func (t *ChainchatChaincode) DeleteUser(stub *shim.ChaincodeStub, args []string) error {
	// Make sure we have the right number of arguments
	if len(args) < 1 {
		errStr := fmt.Sprintf("[ERROR] Expected at least 1 argument to delete a user, received %d", len(args))
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	// Extract information from arguments array
	userID := args[0]

	// Get the user from the ledger
	user, userErr := t.getUser(stub, userID)
	if userErr != nil {
		errStr := fmt.Sprintf("[ERROR] Could not retrieve user %s from the ledger: %s", userID, userErr.Error())
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	// Extract the user's room
	roomID := user.Room

	// Remove the user from the ledger
	deleteErr := stub.DeleteRow("rooms", []shim.Column{
		{&shim.Column_String_{String_: "room"}},
		{&shim.Column_String_{String_: roomID}},
	})

	// Deal with deletion errors
	if deleteErr != nil {
		errStr := fmt.Sprintf("[ERROR] Could not remove user %s from ledger: %s", userID, deleteErr.Error())
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	// Remove the user from the room ledger
	roomErr := t.removeUserFromRoom(stub, userID, roomID)
	if roomErr != nil {
		errStr := fmt.Sprintf("[ERROR] Could not remove user %s from room %s: %s", userID, roomID, roomErr.Error())
		fmt.Println(errStr)
		return errors.New(errStr)
	}

	return nil
}

// Removes a user from a room's list of users
func (t *ChainchatChaincode) removeUserFromRoom(stub *shim.ChaincodeStub, userID string, roomID string) error {
	// Get the room
	room, roomErr := t.getRoom(stub, roomID)
	if roomErr != nil {
		return roomErr
	}

	// Remove the given user ID from the users array
	usersArr := room.UsersIn
	index := indexOf(usersArr, userID)
    usersArr[len(usersArr) - 1], usersArr[index] = usersArr[index], usersArr[len(usersArr) - 1]
    usersArr = usersArr[:len(usersArr) - 1]

    // Set the users array
    room.UsersIn = usersArr

    // Update the room in the ledger
    roomReplaceErr := t.replaceRoom(stub, room)
    if roomReplaceErr != nil {
    	return roomReplaceErr
    }

    return nil
}

// Adds a user to a room's list of users
func (t *ChainchatChaincode) addUserToRoom(stub *shim.ChaincodeStub, userID string, roomID string) error {
	// Get the room
	room, roomErr := t.getRoom(stub, roomID)
	if roomErr != nil {
		return roomErr
	}

	// Add the user to the room's user array
	usersArr := room.UsersIn
	usersArr = append(usersArr, userID)
	room.UsersIn = usersArr

	// Update the room in the ledger
	roomReplaceErr := t.replaceRoom(stub, room)
	if roomReplaceErr != nil {
		return roomReplaceErr
	}

	return nil
}

// Removes a room from the ledger
func (t *ChainchatChaincode) deleteRoom(stub *shim.ChaincodeStub, roomID string) error {
	return stub.DeleteRow("rooms", []shim.Column{
		{&shim.Column_String_{String_: "room"}},
		{&shim.Column_String_{String_: roomID}},
	})
}

// Updates the specified user to be in the specified room
func (t *ChainchatChaincode) updateUserRoom(stub *shim.ChaincodeStub, userID string, roomID string) error {
	// Retrieve the specified user
	user, userErr := t.getUser(stub, userID)
	if userErr != nil {
		return userErr
	}

	// Update the user's room
	user.Room = roomID

	// Update the user in the ledger
	updateErr := t.replaceUser(stub, user)
	if updateErr != nil {
		return updateErr
	}

	return nil
}

// Replaces the room in the table with the given room
func (t *ChainchatChaincode) replaceRoom(stub *shim.ChaincodeStub, room Room) error {
	// Convert the room into bytes for storage
	roomBytes, marshalErr := json.Marshal(room)
	if marshalErr != nil {
		errStr := fmt.Sprintf("Could not marshal room %s into bytes: %s", room.ID, marshalErr.Error())
		return errors.New(errStr)
	}

	// Attemt to replace the room in the ledger
	roomReplaced, replaceErr := stub.ReplaceRow("rooms", shim.Row{
		Columns: []*shim.Column{
			{&shim.Column_String_{String_: "room"}},
			{&shim.Column_String_{String_: room.ID}},
			{&shim.Column_Bytes{Bytes: roomBytes}},
		},
	})

	// Handle ledger errors
	if replaceErr != nil {
		errStr := fmt.Sprintf("Could not replace room %s: %s", room.ID, replaceErr.Error())
		return errors.New(errStr)
	}

	// Handle room not found error
	if !roomReplaced {
		errStr := fmt.Sprintf("Could not replace room %s: room does not exist", room.ID)
		return errors.New(errStr)
	}

	return nil
}

// Replaces the user in the table with the given user
func (t *ChainchatChaincode) replaceUser(stub *shim.ChaincodeStub, user User) error {
	// Convert the user into bytes for storage
	userBytes, marshalErr := json.Marshal(user)
	if marshalErr != nil {
		errStr := fmt.Sprintf("Could not marshal user %s into bytes: %s", user.ID, marshalErr.Error())
		return errors.New(errStr)
	}

	// Attempt to replace the user in the ledger
	userReplaced, replaceErr := stub.ReplaceRow("users", shim.Row{
		Columns: []*shim.Column{
			{&shim.Column_String_{String_: "user"}},
			{&shim.Column_String_{String_: user.ID}},
			{&shim.Column_Bytes{Bytes: userBytes}},
		},
	})

	if replaceErr != nil {
		errStr := fmt.Sprintf("Could not replace user %s: %s", user.ID, replaceErr.Error())
		return errors.New(errStr)
	}

	if !userReplaced {
		errStr := fmt.Sprintf("Could not replacer user %s: no such user exists")
		return errors.New(errStr)
	}

	return nil
}

// Retrieves a single user from the ledger and processes it into a struct
func (t *ChainchatChaincode) getUser(stub *shim.ChaincodeStub, userID string) (User, error) {
	// Retrieve the user from the ledger
	userRow, userRowErr := stub.GetRow("users", []shim.Column{
		shim.Column{Value: &shim.Column_String_{String_: "user"}},
		shim.Column{Value: &shim.Column_String_{String_: userID}},
	})

	// Handle retrieval errors
	if userRowErr != nil {
		errStr := fmt.Sprintf("Could not retrieve user %s from ledger: %s", userID, userRowErr.Error())
		return User{}, errors.New(errStr)
	}

	// Handle not finding the user
	if len(userRow.Columns) == 0 {
		errStr := fmt.Sprintf("Could not find user %s", userID)
		return User{}, errors.New(errStr)
	}

	user := User{}
	userBytes := t.readBytesSafe(userRow.Columns[UserCol])
	unmarshalErr := json.Unmarshal(userBytes, &user)
	if unmarshalErr != nil {
		errStr := fmt.Sprintf("Could not unmarshal user %s: %s", userID, unmarshalErr.Error())
		return User{}, errors.New(errStr)
	}

	return user, nil
}

// Retrieves a particular room from the ledger and processes it into a struct
func (t *ChainchatChaincode) getRoom(stub *shim.ChaincodeStub, roomID string) (Room, error) {
	// Retrieve the room
	roomRow, roomRowErr := stub.GetRow("rooms", []shim.Column{
		shim.Column{Value: &shim.Column_String_{String_: "room"}},
		shim.Column{Value: &shim.Column_String_{String_: roomID}},
	})

	// Handle row retrieval errors
	if roomRowErr != nil {
		errStr := fmt.Sprintf("Could not retrieve the room with ID %s to delete: %s", roomID, roomRowErr.Error())
		return Room{}, errors.New(errStr)
	}

	// Handle unknown room
	if len(roomRow.Columns) == 0 {
		errStr := fmt.Sprintf("Could not find room %s to delete", roomID)
		return Room{}, errors.New(errStr)
	}

	// Turn the room back into a struct
	room := Room{}
	roomBytes := t.readBytesSafe(roomRow.Columns[RoomCol])
	unmarshalErr := json.Unmarshal(roomBytes, &room)
	if unmarshalErr != nil {
		errStr := fmt.Sprintf("Could not unmarshal bytes into a room: %s", unmarshalErr.Error())
		return Room{}, errors.New(errStr)
	}

	return room, nil
}

func (t *ChainchatChaincode) getAndIncrementMsgCounter(stub *shim.ChaincodeStub) (uint64, error) {
	// Retrieve the counter value from the ledger
	counterByte, getErr := stub.GetState("counter")
	if getErr != nil {
		errStr := fmt.Sprintf("Could not retrieve the message counter: %s", getErr.Error())
		return 0, errors.New(errStr)
	}

	// Convert the counter value to a number
	counterStr := string(counterByte[:])
	counter, parseErr := strconv.ParseUint(counterStr, 10, 64)
	if parseErr != nil {
		errStr := fmt.Sprintf("Could not parse counter into a number: %s", parseErr.Error())
		return 0, errors.New(errStr)
	}

	// Increment and store the incremented value
	counterInc := counter + 1
	putErr := stub.PutState("counter", []byte(strconv.FormatUint(counterInc, 10)))
	if putErr != nil {
		errStr := fmt.Sprintf("Could not store the incremented counter value: %s", putErr.Error())
		return 0, errors.New(errStr)
	}

	return counter, nil
}

// Query handles querying for chats to a certain user
func (t *ChainchatChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	// if function == "getMsgs" {
	// 	return t.GetMsgs(stub, args)
	// }

	fmt.Println(fmt.Sprintf("[ERROR] Did not recognize query: %s", function))
	return nil, errors.New("Received unknown function query")
}

func (t *ChainchatChaincode) readStringSafe(col *shim.Column) string {
	if col == nil {
		return ""
	}

	return col.GetString_()
}

func (t *ChainchatChaincode) readInt64Safe(col *shim.Column) int64 {
	if col == nil {
		return 0
	}

	return col.GetInt64()
}

func (t *ChainchatChaincode) readUint64Safe(col *shim.Column) uint64 {
	if col == nil {
		return 0
	}

	return col.GetUint64()
}

func (t *ChainchatChaincode) readBoolSafe(col *shim.Column) bool {
	if col == nil {
		return false
	}

	return col.GetBool()
}

func (t *ChainchatChaincode) readBytesSafe(col *shim.Column) []byte {
	if col == nil {
		return []byte{}
	}

	return col.GetBytes()
}

// Finds the index of a particular string in an array of strings
// Returns -1 if no such string was found
func indexOf(arr []string, elem string) int {
	for i := 0; i < len(arr); i++ {
		if strings.Compare(arr[i], elem) == 0 {
			return i
		}
	}

	return -1
}