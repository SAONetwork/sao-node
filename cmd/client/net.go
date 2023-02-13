package main

import (
	"fmt"
	cliutil "sao-node/cmd"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

var netCmd = &cli.Command{
	Name:  "net",
	Usage: "network management",
	Subcommands: []*cli.Command{
		infoCmd,
		tokenGenCmd,
		nodesCmd,
	},
}

var infoCmd = &cli.Command{
	Name:  "info",
	Usage: "get peer info of the gateway",
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		client, closer, err := getSaoClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		resp, err := client.GetPeerInfo(ctx)
		if err != nil {
			return err
		}

		console := color.New(color.FgMagenta, color.Bold)

		fmt.Print("  GateWay   : ")
		console.Println(client.Cfg.Gateway)

		fmt.Print("  Peer Info : ")
		console.Println(resp.PeerInfo)

		address, err := client.GetNodeAddress(ctx)
		if err != nil {
			return err
		}
		fmt.Print("  Node Address : ")
		console.Println(address)

		status, err := client.GetNodeStatus(ctx, address)
		if err != nil {
			return err
		}
		fmt.Print("  Node Status : ")
		console.Println(status)

		return nil
	},
}

var tokenGenCmd = &cli.Command{
	Name:  "token-gen",
	Usage: "generate token to access http file server",
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		client, closer, err := getSaoClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		didManager, _, err := cliutil.GetDidManager(cctx, client.Cfg.KeyName)
		if err != nil {
			return err
		}

		resp, err := client.GenerateToken(ctx, didManager.Id)
		if err != nil {
			return err
		}

		console := color.New(color.FgMagenta, color.Bold)

		fmt.Print("  DID     : ")
		console.Println(didManager.Id)

		fmt.Print("  GateWay : ")
		console.Println(cliutil.Gateway)

		fmt.Print("  Server  : ")
		console.Println(resp.Server)

		fmt.Print("  Token   : ")
		console.Println(resp.Token)

		return nil
	},
}

var nodesCmd = &cli.Command{
	Name:  "list",
	Usage: "list the nodes in SAO Network",
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		client, closer, err := getSaoClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		nodes, err := client.ListNodes(ctx)
		if err != nil {
			return err
		}
		fmt.Println("Node List: ")
		console := color.New(color.FgMagenta, color.Bold)
		for _, node := range nodes {
			fmt.Println("================================================================")
			fmt.Print("  Address        : ")
			console.Println(node.Creator)
			fmt.Print("  Peer           : ")
			console.Println(node.Peer)
			fmt.Print("  Reputation     : ")
			console.Println(node.Reputation)
			fmt.Print("  Status         : ")
			console.Println(node.Status)
			fmt.Print("  LastAliveHeigh : ")
			console.Println(node.LastAliveHeight)
		}

		return nil
	},
}
