package cli

import (
	"blockchain/lib"
	"flag"
	"fmt"
	"os"
	"strconv"
)

// Cli is used for handling cmd line interface to program
type Cli struct {
	bc *blockchain.BlockChain
}

// NewCli constructs handler for managing cmd line interface
func NewCli(bc *blockchain.BlockChain) *Cli {
	return &Cli{bc}
}

// GetBalance calculates balance amt of all unspent outputs for address
func (cli *Cli) GetBalance(address string) {
	var balance int64

	txOut := cli.bc.GetUnspentOutputs(address)
	for _, output := range txOut {
		balance += output.Value
	}

	fmt.Printf("The Balance of %s is %v", address, balance)
}

// Run starts cmd line interface and parses args
func (cli *Cli) Run() {
	// define two cli input modes
	sendCmdSet := flag.NewFlagSet("send", flag.ExitOnError)
	printCmdSet := flag.NewFlagSet("print", flag.ExitOnError)
	getbalanceCmdSet := flag.NewFlagSet("getbalance", flag.ExitOnError)

	sendFrom := sendCmdSet.String("from", "", "Sender of coins.")
	sendTo := sendCmdSet.String("to", "", "Receiver of coins.")
	sendAmt := sendCmdSet.String("amount", "", "Amount to send.")

	getbalanceCmd := getbalanceCmdSet.String("address", "", "Get the balance of address.")

	if len(os.Args) < 2 {
		fmt.Println("subcommand required.")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "send":
		sendCmdSet.Parse(os.Args[2:])
	case "print":
		printCmdSet.Parse(os.Args[2:])
	case "getbalance":
		getbalanceCmdSet.Parse(os.Args[2:])
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

	if sendCmdSet.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmt == "" {
			sendCmdSet.PrintDefaults()
			os.Exit(1)
		}

		amt, _ := strconv.ParseInt(*sendAmt, 10, 64)
		cli.Send(*sendFrom, *sendTo, int(amt))

	}

	if printCmdSet.Parsed() {
		cli.bc.Print()
	}

	if getbalanceCmdSet.Parsed() {
		if *getbalanceCmd == "" {
			getbalanceCmdSet.PrintDefaults()
			os.Exit(1)
		}

		cli.GetBalance(*getbalanceCmd)
	}
}

func (cli *Cli) Send(from, to string, amount int) {

	txn := blockchain.NewUserTransaction(from, to, amount, cli.bc)
	cli.bc.MineBlock([]*blockchain.Transaction{txn})
	fmt.Println("success...")
}
