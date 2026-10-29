package network

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

// PrintNetworkHelp prints available network subcommands.
func PrintNetworkHelp() {
	fmt.Println("Aeitron Global Infrastructure Network Controller")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  aeitron network <subcommand> [args] [flags]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  bgp peers                            List active Anycast BGP peer sessions and BFD states")
	fmt.Println("  bgp advertise <prefix> [flags]       Announce edge Anycast prefix across global BGP mesh")
	fmt.Println("  bgp withdraw <prefix>                Withdraw Anycast prefix from global BGP mesh")
	fmt.Println("  bgp drain <peer-id> [flags]          Execute RFC 8326 Graceful Shutdown on target peer")
	fmt.Println("  bgp validate <prefix> [--sim]        Validate CIDR against RFC 7454 DFZ boundaries and bogons")
	fmt.Println("  bgp rpki <asn> <prefix>              Perform RFC 6811 Route Origin Validation (ROV)")
	fmt.Println("  bgp flap <prefix>                    Record BGP route flap penalty (RFC 2439 - Operator only)")
	fmt.Println("  bgp damping-status <prefix>          Inspect RFC 2439 decayed route damping state")
	fmt.Println("  flow dispatch <src-ip> <src-port> <dst-ip> <dst-port> [proto]")
	fmt.Println("                                       Route 5-tuple flow using Maglev consistent hashing")
	fmt.Println("  maglev stats                         Display Maglev lookup table size and conntrack metrics")
	fmt.Println("  pop list                             List all 4 PoPs with live health and latency metrics")
	fmt.Println("  pop routes                           Inspect multi-region primary and failover route table")
	fmt.Println("  pop probe                            Execute real-time synthetic health probes across all PoPs")
	fmt.Println("  mesh status                          Display full-mesh overlay HUD, active subsea links, and PQC ratchet age")
	fmt.Println("  mesh peers                           Inspect WireGuard endpoints, allowed IPs, and transfer counters")
	fmt.Println("  mesh rekey <peer-id> [--reason <s>]  Force an immediate post-quantum ML-KEM-1024 ratchet rotation")
	fmt.Println("  mesh topology                        Display mesh adjacency and undersea cable transit bindings")
	fmt.Println("  xdp stats                            Inspect line-rate AF_XDP offload and DDoS packet counters")
	fmt.Println("  xdp wireguard                        Inspect ConnectX-6 zero-copy WireGuard rings and Mpps")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --sim                                Allow testbed documentation prefixes in RFC 7454 validation")
	fmt.Println("  --community <tag>                    BGP community attribute (e.g., 64512:1001)")
	fmt.Println("  --med <metric>                       Multi-Exit Discriminator metric value")
	fmt.Println("  --alternative <pop>                  Target alternative PoP for graceful BGP drain")
	fmt.Println("  --dwell <seconds>                    Dwell duration in seconds for graceful BGP drain (default: 30)")
}

// HandleNetworkCommand routes network subcommands to appropriate handlers.
func HandleNetworkCommand(client *Client, args []string) error {
	if len(args) == 0 {
		PrintNetworkHelp()
		return nil
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "bgp":
		return HandleBGPCommand(client, rest)
	case "mesh":
		return HandleMeshCommand(client, rest)
	case "xdp":
		return HandleXDPCommand(client, rest)
	case "flow":
		return handleFlow(client, rest)
	case "maglev":
		return handleMaglev(client, rest)
	case "pop":
		return handlePoP(client, rest)
	case "help", "--help", "-h":
		PrintNetworkHelp()
		return nil
	default:
		return fmt.Errorf("unknown network subcommand %q; run 'aeitron network help'", sub)
	}
}

// PrintBGPHelp prints help for BGP subcommands.
func PrintBGPHelp() {
	fmt.Println("Aeitron Global BGP-4 Anycast Routing Controller")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  aeitron bgp <subcommand> [args] [flags]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  peers                                List all active BGP peering sessions and BFD states")
	fmt.Println("  advertise <prefix> [--community <tag>] [--med <metric>]")
	fmt.Println("                                       Announce Anycast prefix globally")
	fmt.Println("  withdraw <prefix>                    Withdraw Anycast prefix from BGP routing table")
	fmt.Println("  drain <peer-id> [--dwell <sec>] [--alternative <pop>]")
	fmt.Println("                                       Execute RFC 8326 Graceful BGP Shutdown")
	fmt.Println("  validate <prefix> [--sim]            Validate CIDR against RFC 7454 DFZ boundaries")
	fmt.Println("  rpki <asn> <prefix>                  Perform RFC 6811 Route Origin Validation (ROV)")
	fmt.Println("  flap <prefix>                        Record BGP route flap penalty (RFC 2439)")
	fmt.Println("  damping-status <prefix>              Inspect RFC 2439 decayed route damping state")
	fmt.Println()
}

// HandleBGPCommand handles 'aeitron bgp ...' and 'aeitron network bgp ...' subcommands.
func HandleBGPCommand(client *Client, args []string) error {
	if len(args) == 0 {
		PrintBGPHelp()
		return nil
	}

	action := args[0]
	rest := args[1:]

	switch action {
	case "peers", "list":
		return handleBGPPeers(client, rest)
	case "advertise":
		return handleBGPAdvertise(client, rest)
	case "withdraw":
		return handleBGPWithdraw(client, rest)
	case "drain":
		return handleBGPDrain(client, rest)
	case "validate":
		return handleBGPValidate(client, rest)
	case "rpki":
		return handleBGRPKI(client, rest)
	case "flap":
		return handleBGPFlap(client, rest)
	case "damping-status":
		return handleBGPDampingStatus(client, rest)
	case "help", "--help", "-h":
		PrintBGPHelp()
		return nil
	default:
		return fmt.Errorf("unknown bgp action %q; valid actions: peers, advertise, withdraw, drain, validate, rpki, flap, damping-status", action)
	}
}

func handleBGPPeers(client *Client, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	peers, err := client.ListBGPPeers(ctx)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PEER ADDRESS\tREMOTE ASN\tSTATE\tHOLD TIME\tPREFIXES\tBFD\tREGION")
	for _, p := range peers {
		region := p.Region
		if region == "" {
			region = "global"
		}
		fmt.Fprintf(w, "%s\tAS%d\t%s\t%ds\t%d\t%s\t%s\n",
			p.PeerAddr, p.RemoteASN, p.State, p.HoldTimeSec, p.PrefixesHeld, p.BFDStatus, region)
	}
	return w.Flush()
}

func handleBGPAdvertise(client *Client, args []string) error {
	fs := flag.NewFlagSet("bgp advertise", flag.ContinueOnError)
	comm := fs.String("community", "", "BGP community tag (e.g., 64512:1001)")
	med := fs.Uint("med", 0, "Multi-Exit Discriminator metric")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rem := fs.Args()
	if len(rem) == 0 {
		return fmt.Errorf("prefix argument required; usage: aeitron bgp advertise <prefix> [--community <tag>] [--med <metric>]")
	}
	prefix := rem[0]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	route, err := client.AdvertiseRoute(ctx, prefix, *comm, uint32(*med))
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PREFIX\tSTATUS\tMED\tCOMMUNITIES\tNEXT HOP")
	commsStr := strings.Join(route.Communities, ", ")
	if commsStr == "" {
		commsStr = "-"
	}
	fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
		route.Prefix, route.Status, route.MED, commsStr, route.NextHop)
	return w.Flush()
}

func handleBGPWithdraw(client *Client, args []string) error {
	fs := flag.NewFlagSet("bgp withdraw", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	rem := fs.Args()
	if len(rem) == 0 {
		return fmt.Errorf("prefix argument required; usage: aeitron bgp withdraw <prefix>")
	}
	prefix := rem[0]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	route, err := client.WithdrawRoute(ctx, prefix)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PREFIX\tSTATUS\tACTION")
	fmt.Fprintf(w, "%s\t%s\tWithdrawn from Global Anycast BGP Mesh\n",
		route.Prefix, route.Status)
	return w.Flush()
}

func handleBGPValidate(client *Client, args []string) error {
	fs := flag.NewFlagSet("bgp validate", flag.ContinueOnError)
	sim := fs.Bool("sim", false, "allow testbed prefixes")
	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return fmt.Errorf("prefix argument required; usage: aeitron network bgp validate <prefix> [--sim]")
	}
	prefix := remaining[0]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.ValidatePrefix(ctx, prefix, *sim)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PREFIX\tSTATUS\tSTANDARD\tDETAILS")
	if resp.Status == "VALID" {
		fmt.Fprintf(w, "%s\t%s\tRFC 7454 / DFZ\tAccepted into Global Routing Table\n", resp.Prefix, resp.Status)
	} else {
		fmt.Fprintf(w, "%s\t%s\tRFC 7454 / DFZ\t%s\n", resp.Prefix, resp.Status, resp.Error)
	}
	return w.Flush()
}

func handleBGRPKI(client *Client, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("asn and prefix arguments required; usage: aeitron network bgp rpki <asn> <prefix>")
	}

	asnVal, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		return fmt.Errorf("invalid ASN: %w", err)
	}
	prefix := args[1]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.ValidateRPKI(ctx, prefix, uint32(asnVal))
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PREFIX\tORIGIN ASN\tROV STATUS\tSTANDARD")
	fmt.Fprintf(w, "%s\tAS%d\t%s\tRFC 6811 / RPKI\n", resp.Prefix, resp.ASN, resp.Status)
	return w.Flush()
}

func handleBGPFlap(client *Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("prefix argument required; usage: aeitron network bgp flap <prefix>")
	}
	prefix := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.RecordFlap(ctx, prefix)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PREFIX\tSTATE\tPENALTY\tSTANDARD")
	fmt.Fprintf(w, "%s\t%s\t%.1f\tRFC 2439 Flap Damping\n", resp.Prefix, resp.State, resp.Penalty)
	return w.Flush()
}

func handleBGPDampingStatus(client *Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("prefix argument required; usage: aeitron network bgp damping-status <prefix>")
	}
	prefix := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetDampingStatus(ctx, prefix)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PREFIX\tSTATE\tDECAYED PENALTY\tSTANDARD")
	fmt.Fprintf(w, "%s\t%s\t%.1f\tRFC 2439 Flap Damping\n", resp.Prefix, resp.State, resp.Penalty)
	return w.Flush()
}

func handleBGPDrain(client *Client, args []string) error {
	fs := flag.NewFlagSet("bgp drain", flag.ContinueOnError)
	alt := fs.String("alternative", "pop-sin-gw-02", "target alternative PoP")
	dwell := fs.Int("dwell", 30, "dwell duration in seconds")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rem := fs.Args()
	if len(rem) == 0 {
		return fmt.Errorf("peer-id argument required; usage: aeitron network bgp drain <peer-id> [--alternative <pop>] [--dwell <seconds>]")
	}
	peerID := rem[0]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GenerateDrainPlan(ctx, peerID, *alt, *dwell)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PEER ID\tCOMMUNITY\tLOCAL PREF\tMED\tDWELL\tTARGET ALT")
	fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\t%s\n",
		resp.PeerID, resp.CommunityTag, resp.LocalPreference, resp.MED, resp.DwellDuration, resp.TargetAlternative)
	return w.Flush()
}

func handleFlow(client *Client, args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: aeitron network flow dispatch <src-ip> <src-port> <dst-ip> <dst-port> [proto]")
		return nil
	}

	if args[0] != "dispatch" {
		return fmt.Errorf("unknown flow action %q; valid action: dispatch", args[0])
	}

	rem := args[1:]
	if len(rem) < 4 {
		return fmt.Errorf("insufficient arguments; usage: aeitron network flow dispatch <src-ip> <src-port> <dst-ip> <dst-port> [proto]")
	}

	srcIP := rem[0]
	srcPort, err := strconv.ParseUint(rem[1], 10, 16)
	if err != nil {
		return fmt.Errorf("invalid src-port: %w", err)
	}
	dstIP := rem[2]
	dstPort, err := strconv.ParseUint(rem[3], 10, 16)
	if err != nil {
		return fmt.Errorf("invalid dst-port: %w", err)
	}

	var proto uint8 = 6 // TCP
	if len(rem) >= 5 {
		p, err := strconv.ParseUint(rem[4], 10, 8)
		if err == nil && p > 0 {
			proto = uint8(p)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.DispatchFlow(ctx, srcIP, uint16(srcPort), dstIP, uint16(dstPort), proto)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "FLOW\tASSIGNED BACKEND\tENDPOINT\tWEIGHT")
	flowStr := fmt.Sprintf("%s:%d -> %s:%d (proto:%d)", srcIP, srcPort, dstIP, dstPort, proto)
	endpointStr := fmt.Sprintf("%s:%d", resp.BackendIP, resp.BackendPort)
	fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", flowStr, resp.BackendID, endpointStr, resp.Weight)
	return w.Flush()
}

func handleMaglev(client *Client, args []string) error {
	if len(args) == 0 || args[0] != "stats" {
		fmt.Println("Usage: aeitron network maglev stats")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetMaglevStats(ctx)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RING SIZE (M)\tBACKENDS\tACTIVE CONNTRACK FLOWS\tALGORITHM")
	fmt.Fprintf(w, "%d\t%d\t%d\tGoogle Maglev Consistent Hashing\n",
		resp.TableSize, resp.BackendCount, resp.ConntrackEntries)
	return w.Flush()
}

func handlePoP(client *Client, args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: aeitron network pop <list|routes|probe> [args]")
		return nil
	}

	action := args[0]
	switch action {
	case "list":
		return handlePoPList(client)
	case "routes":
		return handlePoPRoutes(client)
	case "probe":
		return handlePoPProbe(client)
	default:
		return fmt.Errorf("unknown pop action %q; valid actions: list, routes, probe", action)
	}
}

func handlePoPList(client *Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pops, err := client.ListPoPHealth(ctx)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "POP ID\tREGION\tTARGET ENDPOINT\tSTATUS\tHEALTHY\tLAST RTT\tEWMA RTT\tFAILOVER")
	for _, p := range pops {
		failover := p.FailoverPoPID
		if failover == "" {
			failover = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%t\t%.1f ms\t%.1f ms\t%s\n",
			p.PoPID, p.Region, p.TargetEndpoint, p.Status, p.Healthy, p.LastRTTMs, p.EWMALatencyMs, failover)
	}
	return w.Flush()
}

func handlePoPRoutes(client *Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	routes, err := client.GetPoPRoutes(ctx)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "POP ID\tREGION\tPRIMARY TARGET\tSLA RTT\tACTIVE TARGET\tSTATUS\tFAILOVER HOPS")
	for _, r := range routes {
		var hops []string
		for _, h := range r.FailoverTargets {
			resTag := ""
			if h.SovereigntyRestricted {
				resTag = " [SOVEREIGNTY_LOCKED]"
			}
			hops = append(hops, fmt.Sprintf("%s (%s, %.0fms)%s", h.TargetPoP, h.SubseaCable, h.SLARTTMs, resTag))
		}
		failoverSummary := strings.Join(hops, "; ")
		fmt.Fprintf(w, "%s\t%s\t%s\t%.1f ms\t%s\t%s\t%s\n",
			r.PoPID, r.Region, r.PrimaryTarget, r.PrimaryRTTMs, r.ActiveTarget, r.Status, failoverSummary)
	}
	return w.Flush()
}

func handlePoPProbe(client *Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := client.TriggerPoPProbe(ctx)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "POP ID\tTARGET ENDPOINT\tPROBE RESULT\tLATENCY\tHTTP CODE\tERROR")
	for popID, res := range results {
		resStr := "PASS"
		if !res.Success {
			resStr = "FAIL"
		}
		errStr := res.Error
		if errStr == "" {
			errStr = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%.2f ms\t%d\t%s\n",
			popID, res.Target, resStr, res.LatencyMs, res.StatusCode, errStr)
	}
	return w.Flush()
}

// PrintXDPHelp prints help for XDP subcommands.
func PrintXDPHelp() {
	fmt.Println("Aeitron Line-Rate Hardware Offload & AF_XDP Engine")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  aeitron xdp <subcommand> [flags]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  stats                                Display line-rate Mpps, RX/TX packet counters, and DDoS drops")
	fmt.Println("  wireguard                            Inspect Mellanox ConnectX-6 zero-copy WireGuard rings and Mpps")
	fmt.Println()
}

// HandleXDPCommand handles 'aeitron xdp ...' and 'aeitron network xdp ...' subcommands.
func HandleXDPCommand(client *Client, args []string) error {
	if len(args) == 0 {
		PrintXDPHelp()
		return nil
	}

	action := args[0]
	rest := args[1:]

	switch action {
	case "stats":
		return handleXDPStats(client, rest)
	case "wireguard":
		return handleXDPWireGuard(client, rest)
	case "help", "--help", "-h":
		PrintXDPHelp()
		return nil
	default:
		return fmt.Errorf("unknown xdp subcommand %q; run 'aeitron xdp help'", action)
	}
}

func handleXDPStats(client *Client, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats, err := client.GetXDPStats(ctx)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "INTERFACE\tMODE\tLINE RATE\tRX PACKETS\tTX PACKETS\tDDOS DROPS\tPASSED")
	fmt.Fprintf(w, "%s\t%s\t%.2f Mpps\t%d\t%d\t%d\t%d\n",
		stats.Interface, stats.Mode, stats.LineRateMpps, stats.TotalRxPackets, stats.TotalTxPackets, stats.DDoSDrops, stats.PassedPackets)
	return w.Flush()
}

func handleXDPWireGuard(client *Client, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats, err := client.GetWireGuardXDPStats(ctx)
	if err != nil {
		return err
	}

	fmt.Println("====================================================================================================")
	fmt.Println("AEITRON AF_XDP HARDWARE-ACCELERATED WIREGUARD ZERO-COPY OFFLOAD (CONNECTX-6 DX)")
	fmt.Println("====================================================================================================")
	fmt.Printf("NIC Interface:    %s\n", stats.NICInterface)
	fmt.Printf("Driver Mode:      %s\n", stats.DriverMode)
	fmt.Printf("Hardware Offload: %t (Queue ID: %d)\n", stats.HardwareOffload, stats.RXQueueID)
	fmt.Printf("Throughput Rate:  %.2f Mpps (Line-rate saturated zero-copy path)\n", stats.LineRateMpps)
	fmt.Println("----------------------------------------------------------------------------------------------------")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RX PACKETS\tTX PACKETS\tZERO-COPY BYPASS\tREPLAY DROPS\tMALFORMED DROPS\tACTIVE PEERS")
	fmt.Fprintln(w, "----------\t----------\t----------------\t------------\t---------------\t------------")
	fmt.Fprintf(w, "%d\t%d\t%d\t%d\t%d\t%d\n",
		stats.TotalRxPackets, stats.TotalTxPackets, stats.TotalZeroCopyPackets,
		stats.TotalReplayDrops, stats.TotalDroppedPackets, stats.ActivePeersCount)
	w.Flush()
	fmt.Println("====================================================================================================")
	return nil
}
