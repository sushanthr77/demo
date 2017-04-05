package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"os"
	"time"
)

var logger = shim.NewLogger("DCTChaincode")

//==============================================================================================================================
//	 Constants
//==============================================================================================================================
//WorldState
const WS_CARGO_ENROUTE = "CRGENR"
const WS_DOCS_UPLOADED_BY_IMP_BANK = "DUPBIM"
const WS_TRADE_VERIFICATION_SUCCESS = "DOCVSDC"
const WS_TRADE_VERIFICATION_FAILURE = "DOCVFDC"

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
	ParticipantInt
	RelationshipType string    `json:"relationshipType"`
	EnrolDTTM        time.Time `json:"enrolDTTM"`
}

type TradeDoc struct {
	DocumentInt
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

//==============================================================================================================================
//	 Chaincode Methods - Trade Entity
//==============================================================================================================================
func (t *SimpleChaincode) create_trade(stub shim.ChaincodeStubInterface, caller string, caller_affiliation string, trade_json string) ([]byte, error) {

	var trade Trade

	err := json.Unmarshal([]byte(trade_json), &trade) // Convert the JSON defined above into a vehicle object for go

	if err != nil {
		return nil, errors.New("Invalid JSON object provided for create_trade")
	}

	if trade.TradeId == "" || trade.Description == "" || (trade.CreateDTTM == time.Time{}) || trade.ExtRefNum == "" {

		fmt.Printf("CREATE_TRADE: Null value provided for Trade attribute(s)")
		return nil, errors.New("Null value provided for Trade attribute(s)")
	}

	record, err := stub.GetState(trade.TradeId) // If not an error then a record exists so cant create a new trade with this tradeId as it must be unique

	if record != nil {
		return nil, errors.New("Trade already exists")
	}

	add_trade_state(&trade, WS_CARGO_ENROUTE)

	_, err = t.save_trade(stub, trade)

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

	tradeHolder.TradeId = append(tradeHolder.TradeId, trade.TradeId)

	bytes, err = json.Marshal(tradeHolder)

	if err != nil {
		fmt.Print("Error creating Trade_Holder record")
	}

	err = stub.PutState(MK_TRADE, bytes)

	if err != nil {
		return nil, errors.New("Unable to put the state Trade_Holder")
	}

	return nil, errors.New("Something went wrong in func createTrade(trade_json string)")
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

//==============================================================================================================================
//	 Chaincode Methods - Document Entity
//==============================================================================================================================
func (t *SimpleChaincode) create_trade_document(stub shim.ChaincodeStubInterface, caller string, caller_affiliation string,
	document_json string, tradeId string) ([]byte, error) {

	document, err := createTradeDocument(document_json)

	if err != nil {
		return nil, errors.New("Invalid JSON object provided for create_trade_document")
	}

	err = document.DocumentInt.validate()

	if err != nil {
		return nil, err
	}

	record, err := stub.GetState(document.DocumentInt.getId()) // If not an error then a record exists so cant create a new document with this docId as it must be unique

	if record != nil {
		return nil, errors.New("Document already exists")
	}

	_, err = t.save_document(stub, document.DocumentInt)

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

	docHolder.DocumentId = append(docHolder.DocumentId, document.DocumentInt.getId())

	bytes, err = json.Marshal(docHolder)

	if err != nil {
		fmt.Print("Error creating Document_Holder record")
	}

	err = stub.PutState(MK_DOCUMENT, bytes)

	if err != nil {
		return nil, errors.New("Unable to put the state Document_Holder")
	}

	err = t.add_docToTrade(stub, document, tradeId)

	if err != nil {
		return nil, errors.New("Unable to add Document %s to Trade" + document.DocumentInt.getId())
	}

	return nil, errors.New("Something went wrong in func CREATE_DOC")
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

func (t *SimpleChaincode) add_docToTrade(stub shim.ChaincodeStubInterface, tDoc TradeDoc, tradeId string) error {
	v, err := t.retrieve_trade(stub, tradeId)

	if err != nil {
		return errors.New("add_docToTrade: Failed to retrieve Trade")
	} else {
		v.Docs = append(v.Docs, tDoc)
		add_trade_state(&v, WS_DOCS_UPLOADED_BY_IMP_BANK)
		_, err = t.save_trade(stub, v)

		if err != nil {
			fmt.Printf("add_docToTrade: Error saving changes: %s", err)
			return errors.New("add_docToTrade: Error saving changes")
		}
		return nil
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
//	 Chaincode Methods - Verification & Validation
//==============================================================================================================================
func (t *SimpleChaincode) validate_trade(stub shim.ChaincodeStubInterface, caller string, caller_affiliation string,
	document_json string) ([]byte, error) {

	var vData InvoiceValidationData
	err := json.Unmarshal([]byte(document_json), &vData) // Convert the JSON defined above into a InvoiceValidationData object for go

	if err != nil {
		return nil, errors.New("validate_comm_invoice: Incorrect JSON " + err.Error())
	}

	record, err := stub.GetState(vData.docId) // If not an error then a record exists

	if record == nil {
		return nil, errors.New("Document does not exists")
	}

	var si SummaryInvoice
	err = json.Unmarshal(record, &si)

	if err != nil {
		return nil, errors.New("validate_comm_invoice: Could not unmarsal document" + err.Error())
	}

	trade, err2 := t.retrieve_trade(stub, vData.tradeId)
	if err2 != nil {
		fmt.Printf("validate_comm_invoice: Invalid Trade ID: %s", err)
		return nil, err2
	}

	ws := WS_TRADE_VERIFICATION_FAILURE

	if si.TotalAmount == vData.amount {
		ws = WS_TRADE_VERIFICATION_SUCCESS
	}

	_, err2 = add_trade_state(&trade, ws)

	if err2 != nil {
		fmt.Printf("validate_comm_invoice: unable to add world state to Trade: %s", err)
		return nil, errors.New("validate_comm_invoice: unable to add world state to Trade")
	}
	return nil, nil

}

//==============================================================================================================================
//	 Chaincode Methods - Participant Entity
//==============================================================================================================================
func (t *SimpleChaincode) create_participant(stub shim.ChaincodeStubInterface, caller string, caller_affiliation string,
	participant_json string, partyType string) ([]byte, error) {

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

	_, err = t.save_participant(stub, participant)

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

	return nil, errors.New("Something went wrong in func CREATE_Participant")
}

// save_participant - Writes to the ledger the Participant struct passed in a JSON format. Uses the shim file's
//				  method 'PutState'.
func (t *SimpleChaincode) save_participant(stub shim.ChaincodeStubInterface, party ParticipantInt) (bool, error) {

	bytes, err := json.Marshal(party)

	if err != nil {
		fmt.Printf("SAVE_PARTY: Error converting participant record: %s", err)
		return false, errors.New("SAVE_PARTY: Error converting participant record")
	}

	err = stub.PutState(party.getId(), bytes)

	if err != nil {
		fmt.Printf("SAVE_PARTY: Error storing participant record: %s", err)
		return false, errors.New("SAVE_PARTY: Error storing participant record")
	}

	return true, nil
}

func (party Participant) validate() error {
	if party.ParticipantID == "" || party.PrimaryName == "" || party.Address == "" || party.Country == "" {

		fmt.Printf("Validate: Null value provided for Participant attribute(s)")
		return errors.New("Validate: Null value provided for Participant attribute(s)")
	} else {
		return nil
	}
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

	//Trade_Holder
	//	var trades Trade_Holder
	//
	//	bytes, err := json.Marshal(trades)
	//
	//	if err != nil {
	//		return nil, errors.New("Error creating Trade_Holder record")
	//	}
	//
	//	err = stub.PutState(MK_TRADE, bytes)
	//
	//	//Document_Holder
	//	var docs Document_Holder
	//
	//	bytes, err = json.Marshal(docs)
	//
	//	if err != nil {
	//		return nil, errors.New("Error creating Document_Holder record")
	//	}
	//
	//	err = stub.PutState(MK_DOCUMENT, bytes)
	//
	//	//Participant_Holder
	//	var participants Participant_Holder
	//
	//	bytes, err = json.Marshal(participants)
	//
	//	if err != nil {
	//		return nil, errors.New("Error creating Participant_Holder record")
	//	}
	//
	//	err = stub.PutState(MK_PARTICIPANT, bytes)

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

	arg0, err := base64.StdEncoding.DecodeString(args[0])

	if err != nil {
		return nil, errors.New("Error retrieving caller information")
	}

	if function == "create_trade" {
		return t.create_trade(stub, caller, caller_affiliation, string(arg0[0]))
	} else if function == "create_trade_document" {
		return t.create_trade_document(stub, caller, caller_affiliation, args[0], args[1])
	} else if function == "create_participant" {
		return t.create_participant(stub, caller, caller_affiliation, string(arg0[0]), args[1])
	} else if function == "validate_trade" {
		return t.validate_trade(stub, caller, caller_affiliation, args[0])
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

func createTradeDocument(document_json string) (TradeDoc, error) {
	var tDoc TradeDoc
	err := json.Unmarshal([]byte(document_json), &tDoc) // Convert the JSON defined above into a TradeDoc object for go
	if err != nil {
		return tDoc, errors.New("createTradeDocument: Incorrect JSON " + err.Error())
	} else {
		return tDoc, nil
	}
}

func createParticipantFactory(participant_json string, partyType string) (ParticipantInt, error) {
	switch partyType {
	case PT_BANK:
		var bnk Bank
		err := json.Unmarshal([]byte(participant_json), &bnk) // Convert the JSON defined above into a Bank object for go
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

//==============================================================================================================================
//	 MAIN
//==============================================================================================================================
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}

	var ba []byte
	os.Stdout.Write(ba)
	//	b := getSampleTradeList()
	//	os.Stdout.Write(b)

	//	b := getSampleParticipant()
	//	os.Stdout.Write(b)

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
