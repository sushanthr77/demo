package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	//"github.com/satori/go.uuid"
	//	"os"
	"time"
)

var logger = shim.NewLogger("DCTChaincode")

//==============================================================================================================================
//	 Constants
//==============================================================================================================================
//WorldState
const WS_CARGO_ENROUTE = "CRGENR"
const WS_DOCS_UPLOADED = "DOCSUPL"
const WS_TRADE_DECLARED = "TRDDECL"
const WS_TRADE_CLEARED = "TRDCLRD"

//TradeRelationship
const TR_AUTHORITY = "RGLTR"
const TR_IMP_BANK = "IMPBNK"
const TR_EXP_BANK = "EXPBNK"
const TR_IMPORTER = "IMPRTR"
const TR_EXPORTER = "EXPRTR"
const TR_DEST_PORT = "DSTPRT"
const TR_ORGN_PORT = "ORGPRT"
const TR_TRNST_PORT = "TRNSPRT"
const TR_SRC_CUSTOMS = "SRCCSTM"
const TR_DST_CUSTOMS = "DSTCSTM"

//ParticipantType
const PT_AUTHORITY = "RGLTR"
const PT_TRADER = "TRDR"
const PT_PORT = "PORT"
const PT_CUSTOMS = "CSTM"
const PT_BANK = "BANK"

//DocumentType
const DT_SMRY_INVOICE = "SMRYINVC"

//DocumentStatus
const DS_VERIFIED = true
const DS_UNVERIFIED = false

//Country
const CT_UAE = "UAE"
const CT_CHINA = "CHINA"
const CT_INDIA = "INDIA"
const CT_USA = "USA"
const CT_UK = "UK"

//Global Map Keys
const MK_TRADE = "KEY_TRADE"
const MK_DOCUMENT = "KEY_DOCUMENT"
const MK_PARTICIPANT = "KEY_PARTICIPANT"

//==============================================================================================================================
//	 Interface Definitions
//==============================================================================================================================
type DocumentInt interface {
	getType() string
	getId() string
	validate() error
	getDocument() Document
}

type ParticipantInt interface {
	getType() string
	getId() string
	validate() error
}

//==============================================================================================================================
//	 Structure Definitions
//==============================================================================================================================
//	Chaincode - A blank struct for use with Shim (A HyperLedger included go file used for get/put state
//				and other HyperLedger functions)
//==============================================================================================================================
type SimpleChaincode struct {
}

type Trade struct {
	TradeId      string             `json:"tradeId"`
	Description  string             `json:"description"`
	CreateDTTM   time.Time          `json:"createDTTM"`
	ExtRefNum    string             `json:"extRefNum"`
	States       []TradeState       `json:"states"`
	Participants []TradeParticipant `json:"participants"`
	Docs         []TradeDoc         `json:"docs"`
}

type TradeState struct {
	State     string    `json:"state"`
	StateDTTM time.Time `json:"stateDTTM"`
}

type TradeParticipant struct {
	ParticipantID    string    `json:"participantId"`
	RelationshipType string    `json:"relationshipType"`
	EnrolDTTM        time.Time `json:"enrolDTTM"`
}

type TradeDoc struct {
	DocId       string    `json:"docId"`
	AddedBy     string    `json:"addedBy"`
	AddedByType string    `json:"addedByType"`
	AddedDTTM   time.Time `json:"attachDTTM"`
}

type Trade_List struct {
	Trades []Trade `json:"trades"`
}

type Document struct {
	DocId         string    `json:"docId"`
	Type          string    `json:"type"`
	Description   string    `json:"description"`
	CreatedBy     string    `json:"createdBy"`
	CreatedByType string    `json:"createdByType"`
	CreateDTTM    time.Time `json:"createDTTM"`
	ExtRefNum     string    `json:"extRefNum"`
}

type SummaryInvoice struct {
	Document    `json:"document"`
	TotalAmount int64 `json:"totalAmount"`
}

type Participant struct {
	ParticipantID string `json:"participantId"`
	PrimaryName   string `json:"primaryName"`
	Address       string `json:"address"`
	Country       string `json:"country"`
	Type          string `json:"type"`
}

type Trader struct {
	Participant
}

type Bank struct {
	Participant
}

type Port struct {
	Participant
}

type Customs struct {
	Participant
}

type InvoiceValidationData struct {
	tradeId string `json: "tradeId"`
	docId   string `json: "docId"`
	docType string `json: "docType"`
	amount  int64  `json: "amount"`
}

//==============================================================================================================================
//	 Structure Definitions - Global Holders
//==============================================================================================================================
type Trade_Holder struct {
	TradeId []string `json:"tradeIdList"`
}

type Participant_Holder struct {
	ParticipantId []string `json:"participantIdList"`
}

type Document_Holder struct {
	DocumentId []string `json:"documentIdList"`
}

//==============================================================================================================================
//	 Interface Methods
//==============================================================================================================================
func (p Trader) getType() string {
	return p.Type
}

func (p Trader) getId() string {
	return p.ParticipantID
}

func (p Bank) getType() string {
	return p.Type
}

func (p Bank) getId() string {
	return p.ParticipantID
}

func (p Port) getType() string {
	return p.Type
}

func (p Port) getId() string {
	return p.ParticipantID
}

func (p Customs) getType() string {
	return p.Type
}

func (p Customs) getId() string {
	return p.ParticipantID
}

func (sd SummaryInvoice) getType() string {
	return sd.Type
}

func (sd SummaryInvoice) getId() string {
	return sd.DocId
}

func (sd SummaryInvoice) getDocument() Document {
	return sd.Document
}

func (sd Document) getType() string {
	return sd.Type
}

func (sd Document) getId() string {
	return sd.DocId
}

func (party Trader) validate() error {
	return party.Participant.validate()
}

func (party Customs) validate() error {
	return party.Participant.validate()
}

func (party Bank) validate() error {
	return party.Participant.validate()
}

func (party Port) validate() error {
	return party.Participant.validate()
}

func (party Participant) validate() error {
	if party.ParticipantID == "" || party.PrimaryName == "" || party.Address == "" || party.Country == "" {

		fmt.Printf("Validate: Null value provided for Participant attribute(s)")
		return errors.New("Validate: Null value provided for Participant attribute(s)")
	} else {
		return nil
	}
}

//==============================================================================================================================
//	 Chaincode Methods - Trade Entity
//==============================================================================================================================
func (t *SimpleChaincode) create_trade(stub shim.ChaincodeStubInterface, caller string, caller_affiliation string, trade_json []byte) ([]byte, error) {

	var trades Trade_List

	err := json.Unmarshal(trade_json, &trades) // Convert the JSON defined above into a vehicle object for go

	if err != nil {
		return nil, errors.New("Invalid JSON object provided for create_trade")
	}

	logger.Debug(trades.Trades[0].TradeId + trades.Trades[0].Description + trades.Trades[0].CreateDTTM.String() + trades.Trades[0].ExtRefNum)

	if trades.Trades[0].TradeId == "" || trades.Trades[0].Description == "" || trades.Trades[0].CreateDTTM.String() == "" || trades.Trades[0].ExtRefNum == "" {

		fmt.Printf("CREATE_TRADE: Null value provided for Trade attribute(s)")
		return nil, errors.New("Null value provided for Trade attribute(s)")
	}

	record, err := stub.GetState(trades.Trades[0].TradeId) // If not an error then a record exists so cant create a new trade with this tradeId as it must be unique

	if record != nil {
		return nil, errors.New("Trade already exists")
	}

	add_trade_state(&trades.Trades[0], WS_CARGO_ENROUTE)

	_, err = t.save_trade(stub, trades.Trades[0])

	if err != nil {
		fmt.Printf("CREATE_TRADE: Error saving changes: %s", err)
		return nil, errors.New("Error saving changes")
	}

	bytes, err := stub.GetState(MK_TRADE)

	if err != nil {
		return nil, errors.New("func createTrade(trade_json string) unable to get tradeId")
	}

	var tradeHolder Trade_Holder

	err = json.Unmarshal(bytes, &tradeHolder)

	if err != nil {
		return nil, errors.New("Corrupt Trade_Holder record")
	}

	tradeHolder.TradeId = append(tradeHolder.TradeId, trades.Trades[0].TradeId)

	bytes, err = json.Marshal(tradeHolder)

	if err != nil {
		fmt.Print("Error creating Trade_Holder record")
	}

	err = stub.PutState(MK_TRADE, bytes)

	if err != nil {
		return nil, errors.New("Unable to put the state Trade_Holder")
	}

	return nil, nil
}

// save_trade - Writes to the ledger the Trade struct passed in a JSON format. Uses the shim file's
//				  method 'PutState'.
func (t *SimpleChaincode) save_trade(stub shim.ChaincodeStubInterface, trade Trade) (bool, error) {

	bytes, err := json.Marshal(trade)

	if err != nil {
		fmt.Printf("SAVE_TRADE: Error converting trade record: %s", err)
		return false, errors.New("Error converting trade record")
	}

	err = stub.PutState(trade.TradeId, bytes)

	if err != nil {
		fmt.Printf("SAVE_TRADE: Error storing trade record: %s", err)
		return false, errors.New("Error storing trade record")
	}

	return true, nil
}

//	 get_trades
func (t *SimpleChaincode) get_trades(stub shim.ChaincodeStubInterface, caller string, caller_affiliation string) ([]byte, error) {
	bytes, err := stub.GetState(MK_TRADE)

	if err != nil {
		return nil, errors.New("Unable to get MK_TRADE")
	}

	var trades Trade_Holder

	err = json.Unmarshal(bytes, &trades)

	if err != nil {
		return nil, errors.New("Corrupt Trade_Holder")
	}

	result := "["

	var temp []byte
	var v Trade

	for _, tradeId := range trades.TradeId {

		v, err = t.retrieve_trade(stub, tradeId)

		if err != nil {
			return nil, errors.New("Failed to retrieve Trade")
		}

		temp, err = t.get_trade_details(stub, v, caller, caller_affiliation)

		if err == nil {
			result += string(temp) + ","
		}
	}

	if len(result) == 1 {
		result = "[]"
	} else {
		result = result[:len(result)-1] + "]"
	}

	return []byte(result), nil
}

//	 retrieve_trade - Gets the state of the data at tradeID in the ledger then converts it from the stored
//					JSON into the Trade struct for use in the contract. Returns the Trade struct.
//					Returns empty v if it errors.
func (t *SimpleChaincode) retrieve_trade(stub shim.ChaincodeStubInterface, tradeId string) (Trade, error) {

	var v Trade

	bytes, err := stub.GetState(tradeId)

	if err != nil {
		fmt.Printf("RETRIEVE_TRADE: Failed to invoke tradeId: %s", err)
		return v, errors.New("RETRIEVE_TRADE: Error retrieving trade with ID = " + tradeId)
	}

	err = json.Unmarshal(bytes, &v)

	if err != nil {
		fmt.Printf("RETRIEVE_TRADE: Corrupt trade record "+string(bytes)+": %s", err)
		return v, errors.New("RETRIEVE_TRADE: Corrupt trade record" + string(bytes))
	}

	return v, nil
}

//	 get_trade_details
func (t *SimpleChaincode) get_trade_details(stub shim.ChaincodeStubInterface, v Trade, caller string, caller_affiliation string) ([]byte, error) {

	bytes, err := json.Marshal(v)

	if err != nil {
		return nil, errors.New("GET_TRADE_DETAILS: Invalid trade object")
	}

	return bytes, nil
}

//	 add_trade_state
func (t *SimpleChaincode) add_trade_state(stub shim.ChaincodeStubInterface, caller string, caller_affiliation string, tradeId string, state string) ([]byte, error) {

	v, err := t.retrieve_trade(stub, tradeId)

	if err != nil {
		return nil, errors.New("add_trade_state: Failed to retrieve Trade")
	} else {

		add_trade_state(&v, state)
		_, err = t.save_trade(stub, v)

		if err != nil {
			fmt.Printf("add_trade_state: Error saving changes: %s", err)
			return nil, errors.New("add_trade_state: Error saving changes")
		}
		return nil, nil
	}
}

//==============================================================================================================================
//	 Chaincode Methods - Document Entity
//==============================================================================================================================
func (t *SimpleChaincode) create_document(stub shim.ChaincodeStubInterface, caller string, caller_affiliation string,
	document_json []byte, documentType string) ([]byte, error) {

	document, err := createDocument(document_json, documentType)

	if err != nil {
		return nil, errors.New("Invalid JSON object provided for create_trade_document")
	}

	err = document.validate()

	if err != nil {
		return nil, err
	}

	record, err := stub.GetState(document.getId()) // If not an error then a record exists so cant create a new document with this docId as it must be unique

	if record != nil {
		return nil, errors.New("Document already exists")
	}

	_, err = t.save_document(stub, document)

	if err != nil {
		fmt.Printf("CREATE_DOC: Error saving changes: %s", err)
		return nil, errors.New("CREATE_DOC: Error saving changes")
	}

	bytes, err := stub.GetState(MK_DOCUMENT)

	if err != nil {
		return nil, errors.New("func CREATE_DOC(trade_json string) unable to get docId")
	}

	var docHolder Document_Holder

	err = json.Unmarshal(bytes, &docHolder)

	if err != nil {
		return nil, errors.New("Corrupt Document_Holder record")
	}

	docHolder.DocumentId = append(docHolder.DocumentId, document.getId())

	bytes, err = json.Marshal(docHolder)

	if err != nil {
		fmt.Print("Error creating Document_Holder record")
	}

	err = stub.PutState(MK_DOCUMENT, bytes)

	if err != nil {
		return nil, errors.New("Unable to put the state Document_Holder")
	}

	return nil, nil
}

//==============================================================================================================================
//	 Chaincode Methods - Document Entity
//==============================================================================================================================
//	 retrieve_document
func (t *SimpleChaincode) retrieve_document(stub shim.ChaincodeStubInterface, documentId string) ([]byte, error) {

	bytes, err := stub.GetState(documentId)

	if err != nil {
		fmt.Printf("retrieve_document: Failed to invoke documentId: %s", err)
		return nil, errors.New("retrieve_document: Error documentId trade with ID = " + documentId)
	}

	return bytes, nil
}

//==============================================================================================================================
//	 Chaincode Methods - Document Entity
//==============================================================================================================================
//	 get_participants
func (t *SimpleChaincode) get_documents(stub shim.ChaincodeStubInterface, caller string, caller_affiliation string) ([]byte, error) {
	bytes, err := stub.GetState(MK_DOCUMENT)

	if err != nil {
		return nil, errors.New("Unable to get MK_DOCUMENT")
	}

	var documents Document_Holder

	err = json.Unmarshal(bytes, &documents)

	if err != nil {
		return nil, errors.New("Corrupt Document_Holder")
	}

	result := "["

	var temp []byte

	for _, documentsId := range documents.DocumentId {

		temp, err = t.retrieve_document(stub, documentsId)

		if err != nil {
			return nil, errors.New("Failed to retrieve Document")
		}

		//		temp, err = t.get_participant_details(stub, v, caller, caller_affiliation)

		//		if err == nil {
		result += string(temp) + ","
		//		}
	}

	if len(result) == 1 {
		result = "[]"
	} else {
		result = result[:len(result)-1] + "]"
	}

	return []byte(result), nil
}

// save_document - Writes to the ledger the Document struct passed in a JSON format. Uses the shim file's
//				  method 'PutState'.
func (t *SimpleChaincode) save_document(stub shim.ChaincodeStubInterface, doc DocumentInt) (bool, error) {

	bytes, err := json.Marshal(doc)

	if err != nil {
		fmt.Printf("SAVE_DOC: Error converting document record: %s", err)
		return false, errors.New("SAVE_DOC: Error converting document record")
	}

	err = stub.PutState(doc.getId(), bytes)

	if err != nil {
		fmt.Printf("SAVE_DOC: Error storing document record: %s", err)
		return false, errors.New("Error storing document record")
	}

	return true, nil
}

func (t *SimpleChaincode) add_doc_to_trade(stub shim.ChaincodeStubInterface, caller string, caller_affiliation string, json_data []byte, tradeId string) ([]byte, error) {
	v, err := t.retrieve_trade(stub, tradeId)

	if err != nil {
		return nil, errors.New("add_docToTrade: Failed to retrieve Trade")
	} else {
		var tDoc TradeDoc

		err = json.Unmarshal(json_data, &tDoc)

		if err != nil {
			return nil, errors.New("Corrupt TradeDoc JSON received")
		}

		v.Docs = append(v.Docs, tDoc)
		add_trade_state(&v, WS_DOCS_UPLOADED)
		_, err = t.save_trade(stub, v)

		if err != nil {
			fmt.Printf("add_docToTrade: Error saving changes: %s", err)
			return nil, errors.New("add_docToTrade: Error saving changes")
		}
		return nil, nil
	}

}

func (si SummaryInvoice) validate() error {
	if si.DocId == "" || si.Description == "" || (si.CreateDTTM == time.Time{}) ||
		si.ExtRefNum == "" || si.CreatedBy == "" || si.CreatedByType == "" || si.TotalAmount <= 0 {

		fmt.Printf("CREATE_DOC: Null value provided for Document attribute(s)")
		return errors.New("Null value provided for Document attribute(s)")
	} else {
		return nil
	}
}

//==============================================================================================================================
//	 Chaincode Methods - Participant Entity
//==============================================================================================================================
func (t *SimpleChaincode) create_participant(stub shim.ChaincodeStubInterface, caller string, caller_affiliation string,
	participant_json []byte, partyType string) ([]byte, error) {

	participant, err := createParticipantFactory(participant_json, partyType)

	if err != nil {
		return nil, errors.New("Invalid JSON object provided for create_participant")
	}

	err = participant.validate()

	if err != nil {
		return nil, err
	}

	record, err := stub.GetState(participant.getId()) // If not an error then a record exists so cant create a new participant with this participantId as it must be unique

	if record != nil {
		return nil, errors.New("Participant already exists")
	}

	_, err = t.save_participant(stub, participant_json, participant.getId())

	if err != nil {
		fmt.Printf("CREATE_PARTICIPANT: Error saving changes: %s", err)
		return nil, errors.New("CREATE_PARTICIPANT: Error saving changes")
	}

	bytes, err := stub.GetState(MK_PARTICIPANT)

	if err != nil {
		return nil, errors.New("func CREATE_PARTICIPANT(trade_json string) unable to get participantId")
	}

	var partyHolder Participant_Holder

	err = json.Unmarshal(bytes, &partyHolder)

	if err != nil {
		return nil, errors.New("Corrupt Participant_Holder record")
	}

	partyHolder.ParticipantId = append(partyHolder.ParticipantId, participant.getId())

	bytes, err = json.Marshal(partyHolder)

	if err != nil {
		fmt.Print("Error creating Participant_Holder record")
	}

	err = stub.PutState(MK_PARTICIPANT, bytes)

	if err != nil {
		return nil, errors.New("Unable to put the state Participant_Holder")
	}

	return nil, nil
}

// save_participant - Writes to the ledger the Participant struct passed in a JSON format. Uses the shim file's
//				  method 'PutState'.
func (t *SimpleChaincode) save_participant(stub shim.ChaincodeStubInterface, partyJson []byte, partyId string) (bool, error) {

	//	bytes, err := json.Marshal(party)
	//
	//	if err != nil {
	//		fmt.Printf("SAVE_PARTY: Error converting participant record: %s", err)
	//		return false, errors.New("SAVE_PARTY: Error converting participant record")
	//	}

	err := stub.PutState(partyId, partyJson)

	if err != nil {
		fmt.Printf("SAVE_PARTY: Error storing participant record: %s", err)
		return false, errors.New("SAVE_PARTY: Error storing participant record")
	}

	return true, nil
}

//	 get_participants
func (t *SimpleChaincode) get_participants(stub shim.ChaincodeStubInterface, caller string, caller_affiliation string) ([]byte, error) {
	bytes, err := stub.GetState(MK_PARTICIPANT)

	if err != nil {
		return nil, errors.New("Unable to get MK_PARTICIPANT")
	}

	var participants Participant_Holder

	err = json.Unmarshal(bytes, &participants)

	if err != nil {
		return nil, errors.New("Corrupt Participant_Holder")
	}

	result := "["

	var temp []byte

	for _, participantsId := range participants.ParticipantId {

		temp, err = t.retrieve_participant(stub, participantsId)

		if err != nil {
			return nil, errors.New("Failed to retrieve Participant")
		}

		//		temp, err = t.get_participant_details(stub, v, caller, caller_affiliation)

		//		if err == nil {
		result += string(temp) + ","
		//		}
	}

	if len(result) == 1 {
		result = "[]"
	} else {
		result = result[:len(result)-1] + "]"
	}

	return []byte(result), nil
}

//	 retrieve_participant - Gets the state of the data at participantID in the ledger then converts it from the stored
//					JSON into the Trade struct for use in the contract. Returns the participant struct.
//					Returns empty v if it errors.
func (t *SimpleChaincode) retrieve_participant(stub shim.ChaincodeStubInterface, participantId string) ([]byte, error) {

	bytes, err := stub.GetState(participantId)

	if err != nil {
		fmt.Printf("retrieve_participant: Failed to invoke participantId: %s", err)
		return nil, errors.New("retrieve_participant: Error participantId trade with ID = " + participantId)
	}

	return bytes, nil
}

func (t *SimpleChaincode) add_participant_to_trade(stub shim.ChaincodeStubInterface,
	caller string, caller_affiliation string, json_data []byte, tradeId string) ([]byte, error) {
	v, err := t.retrieve_trade(stub, tradeId)

	if err != nil {
		return nil, errors.New("add_participant_to_trade: Failed to retrieve Trade")
	} else {
		var tParticipant TradeParticipant

		err = json.Unmarshal(json_data, &tParticipant)

		if err != nil {
			return nil, errors.New("Corrupt TradeParticipant JSON received")
		}

		v.Participants = append(v.Participants, tParticipant)
		_, err = t.save_trade(stub, v)

		if err != nil {
			fmt.Printf("add_participant_to_trade: Error saving changes: %s", err)
			return nil, errors.New("add_participant_to_trade: Error saving changes")
		}
		return nil, nil
	}
}

//=================================================================================================================================
//	 Ping Function
//=================================================================================================================================
//	 Pings the peer to keep the connection alive
//=================================================================================================================================
func (t *SimpleChaincode) ping(stub shim.ChaincodeStubInterface) ([]byte, error) {
	return []byte("Hello, world!"), nil
}

//==============================================================================================================================
//	Init Function - Called when the user deploys the chaincode
//==============================================================================================================================
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Printf("\nSUSH: In Init method of chaincode...")
	//Args
	//				0
	//			peer_address

	var trades Trade_Holder

	bytes, err := json.Marshal(trades)

	if err != nil {
		return nil, errors.New("Error creating Trade_Holder record")
	}

	err = stub.PutState(MK_TRADE, bytes)

	//Document_Holder
	var docs Document_Holder

	bytes, err = json.Marshal(docs)

	if err != nil {
		return nil, errors.New("Error creating Document_Holder record")
	}

	err = stub.PutState(MK_DOCUMENT, bytes)

	//Participant_Holder
	var participants Participant_Holder

	bytes, err = json.Marshal(participants)

	if err != nil {
		return nil, errors.New("Error creating Participant_Holder record")
	}

	err = stub.PutState(MK_PARTICIPANT, bytes)

	return nil, nil
}

//=================================================================================================================================
//	Query - Called on chaincode query. Takes a function name passed and calls that function. Passes the
//  		initial arguments passed are passed on to the called function.
//=================================================================================================================================
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	caller, caller_affiliation, err := t.get_caller_data(stub)
	if err != nil {
		fmt.Printf("QUERY: Error retrieving caller details", err)
		return nil, errors.New("QUERY: Error retrieving caller details: " + err.Error())
	}

	logger.Debug("function: ", function)
	logger.Debug("caller: ", caller)
	logger.Debug("affiliation: ", caller_affiliation)

	if function == "get_trades" {
		return t.get_trades(stub, caller, caller_affiliation)
	} else if function == "get_participants" {
		return t.get_participants(stub, caller, caller_affiliation)
	} else if function == "get_documents" {
		return t.get_documents(stub, caller, caller_affiliation)
	}

	return nil, errors.New("Received unknown function invocation " + function)

}

//==============================================================================================================================
//	 Router Functions
//==============================================================================================================================
//	Invoke - Called on chaincode invoke. Takes a function name passed and calls that function. Converts some
//		  initial arguments passed to other things for use in the called function e.g. name -> ecert
//==============================================================================================================================
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	caller, caller_affiliation, err := t.get_caller_data(stub)

	arg0, err := decodeBase64(args[0])
	logger.Debug("Undecoded payload = " + args[0])

	if err != nil {
		logger.Debug("Error = " + err.Error())
		return nil, errors.New("Error retrieving caller information")
	}

	if function == "create_trade" {
		return t.create_trade(stub, caller, caller_affiliation, arg0)
	} else if function == "create_document" {
		return t.create_document(stub, caller, caller_affiliation, arg0, args[1])
	} else if function == "create_participant" {
		return t.create_participant(stub, caller, caller_affiliation, arg0, args[1])
	} else if function == "add_doc_to_trade" {
		return t.add_doc_to_trade(stub, caller, caller_affiliation, arg0, args[1])
	} else if function == "add_participant_to_trade" {
		return t.add_participant_to_trade(stub, caller, caller_affiliation, arg0, args[1])
	} else if function == "add_trade_state" {
		return t.add_trade_state(stub, caller, caller_affiliation, args[0], args[1])
	} else if function == "ping" {
		return t.ping(stub)
	}
	return nil, errors.New("Function by name " + function + " doesn't exist.")
}

//==============================================================================================================================
//	 get_caller_data - Calls the get_ecert and check_role functions and returns the ecert and role for the
//					 name passed.
//==============================================================================================================================

func (t *SimpleChaincode) get_caller_data(stub shim.ChaincodeStubInterface) (string, string, error) {

	//	user, err := t.get_username(stub)
	//
	//	affiliation, err := t.check_affiliation(stub)
	//
	//	if err != nil {
	//		return "", "", err
	//	}
	//
	//	return user, affiliation, nil
	return "", "", nil
}

//==============================================================================================================================
//	 get_caller - Retrieves the username of the user who invoked the chaincode.
//				  Returns the username as a string.
//==============================================================================================================================

func (t *SimpleChaincode) get_username(stub shim.ChaincodeStubInterface) (string, error) {

	username, err := stub.ReadCertAttribute("username")
	if err != nil {
		return "", errors.New("Couldn't get attribute 'username'. Error: " + err.Error())
	}
	return string(username), nil
}

//==============================================================================================================================
//	 check_affiliation - Takes an ecert as a string, decodes it to remove html encoding then parses it and checks the
// 				  		certificates common name. The affiliation is stored as part of the common name.
//==============================================================================================================================

func (t *SimpleChaincode) check_affiliation(stub shim.ChaincodeStubInterface) (string, error) {
	affiliation, err := stub.ReadCertAttribute("role")
	if err != nil {
		return "", errors.New("Couldn't get attribute 'role'. Error: " + err.Error())
	}
	return string(affiliation), nil

}

//==============================================================================================================================
//	 Global Methods
//==============================================================================================================================
func add_trade_state(trade *Trade, state string) (*Trade, error) {
	var ts TradeState
	ts.State = state
	ts.StateDTTM = time.Now()
	trade.States = append(trade.States, ts)
	return trade, nil
}

func createDocument(document_json []byte, docType string) (DocumentInt, error) {
	switch docType {
	case DT_SMRY_INVOICE:
		var sInv SummaryInvoice
		err := json.Unmarshal(document_json, &sInv) // Convert the JSON defined above into a SummaryInvoice object for go
		if err != nil {
			return nil, errors.New("createDocument: Incorrect JSON " + err.Error())
		} else {
			return sInv, nil
		}
	default:
		return nil, errors.New("createDocument: Unknown document type specified")

	}
}

func createParticipantFactory(participant_json []byte, partyType string) (ParticipantInt, error) {
	switch partyType {
	case PT_BANK:
		var bnk Bank
		err := json.Unmarshal(participant_json, &bnk) // Convert the JSON defined above into a Bank object for go
		if err != nil {
			return nil, errors.New("Error unmarshalling Participant Bank" + err.Error())
		} else {
			return bnk, nil
		}
	case PT_CUSTOMS:
		var cstm Customs
		err := json.Unmarshal([]byte(participant_json), &cstm) // Convert the JSON defined above into a Customs object for go
		if err != nil {
			return nil, errors.New("Error unmarshalling Participant Customs" + err.Error())
		} else {
			return cstm, nil
		}
	case PT_PORT:
		var prt Port
		err := json.Unmarshal([]byte(participant_json), &prt) // Convert the JSON defined above into a Port object for go
		if err != nil {
			return nil, errors.New("Error unmarshalling Participant Port" + err.Error())
		} else {
			return prt, nil
		}
	case PT_TRADER:
		var trd Trader
		err := json.Unmarshal([]byte(participant_json), &trd) // Convert the JSON defined above into a Trader object for go
		if err != nil {
			return nil, errors.New("Error unmarshalling Participant Trader" + err.Error())
		} else {
			return trd, nil
		}
	default:
		return nil, errors.New("Unknown participant type specified")
	}
}

func decodeBase64(data string) ([]byte, error) {

	arg0, err := base64.StdEncoding.DecodeString(data)

	if err == nil {
		return arg0, nil
	}

	arg0, err = base64.RawStdEncoding.DecodeString(data)

	if err == nil {
		return arg0, nil
	}

	arg0, err = base64.RawURLEncoding.DecodeString(data)

	if err == nil {
		return arg0, nil
	}

	return nil, err
}

//func generatePrimaryKey() string {
//	return uuid.NewV4().String()
//}

//==============================================================================================================================
//	 MAIN
//==============================================================================================================================
func main() {
		err := shim.Start(new(SimpleChaincode))
		if err != nil {
			fmt.Printf("Error starting Simple chaincode: %s", err)
		}
	//
	//	var ba []byte
	//	os.Stdout.Write(ba)

	//	pl := "eyJ0cmFkZXMiOlt7InRyYWRlSWQiOiIxMDAwMDEiLCJkZXNjcmlwdGlvbiI6IlRyYWRlIGJldHdlZW4gU2Ftc3VuZyBLb3JlYSBhbmQgU2Ftc3VuZyBEdWJhaSwgU2hpcG1lbnQgb2YgbW9iaWxlIHBob25lcyIsImNyZWF0ZURUVE0iOiIwMDAxLTAxLTAxVDAwOjAwOjAwWiIsImV4dFJlZk51bSI6IlNORy00MjM5REYtODEwNzQiLCJzdGF0ZXMiOlt7InN0YXRlIjoiQ1JHRU5SIiwic3RhdGVEVFRNIjoiMDAwMS0wMS0wMVQwMDowMDowMFoifSx7InN0YXRlIjoiRFVQQklNIiwic3RhdGVEVFRNIjoiMDAwMS0wMS0wMVQwMDowMDowMFoifSx7InN0YXRlIjoiRE9DVkZEQyIsInN0YXRlRFRUTSI6IjAwMDEtMDEtMDFUMDA6MDA6MDBaIn0seyJzdGF0ZSI6IkRPQ1ZTREMiLCJzdGF0ZURUVE0iOiIwMDAxLTAxLTAxVDAwOjAwOjAwWiJ9XSwicGFydGljaXBhbnRzIjpbeyJwYXJ0aWNpcGFudElkIjoiMTAwIiwicmVsYXRpb25zaGlwVHlwZSI6IkRTVENTVE0iLCJlbnJvbERUVE0iOiIwMDAxLTAxLTAxVDAwOjAwOjAwWiJ9LHsicGFydGljaXBhbnRJZCI6IjIwMCIsInJlbGF0aW9uc2hpcFR5cGUiOiJJTVBSVFIiLCJlbnJvbERUVE0iOiIwMDAxLTAxLTAxVDAwOjAwOjAwWiJ9LHsicGFydGljaXBhbnRJZCI6IjMwMCIsInJlbGF0aW9uc2hpcFR5cGUiOiJJTVBCTksiLCJlbnJvbERUVE0iOiIwMDAxLTAxLTAxVDAwOjAwOjAwWiJ9XSwiZG9jcyI6W3siZG9jSWQiOiIxMjM0NTU2NSIsImFkZGVkQnkiOiIyMDAiLCJhZGRlZEJ5VHlwZSI6IkJBTksiLCJhdHRhY2hEVFRNIjoiMDAwMS0wMS0wMVQwMDowMDowMFoifSx7ImRvY0lkIjoiMjIzNDU1NjUiLCJhZGRlZEJ5IjoiMjAwIiwiYWRkZWRCeVR5cGUiOiJCQU5LIiwiYXR0YWNoRFRUTSI6IjAwMDEtMDEtMDFUMDA6MDA6MDBaIn0seyJkb2NJZCI6IjMyMzQ1NTY1IiwiYWRkZWRCeSI6IjIwMCIsImFkZGVkQnlUeXBlIjoiQkFOSyIsImF0dGFjaERUVE0iOiIwMDAxLTAxLTAxVDAwOjAwOjAwWiJ9XX1dfQ=="
	//
	//	arg0, err := decodeBase64(pl)
	//	os.Stdout.Write(arg0)
	//
	//	if err != nil {
	//		fmt.Printf("\nError1 : %s", err)
	//	}
	//
	//	var trades Trade_List
	//
	//	err = json.Unmarshal(arg0, &trades) // Convert the JSON defined above into a vehicle object for go
	//
	//	if err != nil {
	//		fmt.Printf("\nError2 : %s", err)
	//	}
	//
	//	fmt.Printf("\n" + trades.Trades[0].TradeId + trades.Trades[0].Description + trades.Trades[0].CreateDTTM.String() + trades.Trades[0].ExtRefNum)
	//
	//	if trades.Trades[0].TradeId == "" || trades.Trades[0].Description == "" || (trades.Trades[0].CreateDTTM == time.Time{}) || trades.Trades[0].ExtRefNum == "" {
	//
	//		fmt.Printf("CREATE_TRADE: Null value provided for Trade attribute(s)")
	//	}

	//	b := getSampleTradeList()
	//	os.Stdout.Write(b)

	//	b := getSampleParticipant()
	//	os.Stdout.Write(b)

	//u1 := generatePrimaryKey()
	//fmt.Printf("UUIDv4: %s\n", u1)

	//	document_json := getSampleSummaryInvoice()
	//	os.Stdout.Write(document_json)

	//	document_json := getSampleTradeList()
	//
	//	var td Trade
	//	err := json.Unmarshal(document_json, &td)
	//
	//	if err != nil {
	//		fmt.Printf("\nError unmarshalling trade json: %s", err)
	//	}
	//
	//	_, err = add_trade_state(&td, WS_CARGO_ENROUTE)
	//
	//	if err != nil {
	//		fmt.Printf("\nError in add_trade_state: %s", err)
	//	} else {
	//		fmt.Printf("\nSuccess")
	//		document_json, err = json.Marshal(td)
	//		os.Stdout.Write(document_json)
	//	}
	//	d := getSampleTradeList()
	//	os.Stdout.Write(d)

	//		err := shim.Start(new(SimpleChaincode))
	//
	//		if err != nil {
	//			fmt.Printf("Error starting Chaincode: %s", err)
	//		}
}
