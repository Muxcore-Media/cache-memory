package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	cachememory "github.com/Muxcore-Media/cache-memory"
	modulev1 "github.com/Muxcore-Media/core/proto/gen/muxcore/module/v1"
)

func main() {
	meshAddr := flag.String("muxcore-mesh-addr", "localhost:9700", "gRPC address of the core mesh")
	moduleID := flag.String("muxcore-module-id", "cache-memory", "unique module identifier")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("sidecar starting: mesh=%s module=%s", *meshAddr, *moduleID)

	// 1. Read muxcore.json for registration info
	infoJSON, err := os.ReadFile("muxcore.json")
	if err != nil {
		log.Fatalf("failed to read muxcore.json: %v", err)
	}

	// Compact the JSON (trim whitespace)
	var rawInfo map[string]any
	if err := json.Unmarshal(infoJSON, &rawInfo); err != nil {
		log.Fatalf("failed to parse muxcore.json: %v", err)
	}
	compactJSON, err := json.Marshal(rawInfo)
	if err != nil {
		log.Fatalf("failed to marshal muxcore.json: %v", err)
	}

	// 2. Create the module using the existing factory
	mod := cachememory.NewModule()
	modInfo := mod.Info()
	modInfo.ID = *moduleID

	// 3. Connect to the core gRPC mesh
	conn, err := grpc.Dial(*meshAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("failed to connect to core mesh at %s: %v", *meshAddr, err)
	}
	defer conn.Close()

	client := modulev1.NewModuleRegistrationClient(conn)
	ctx := context.Background()

	// 4. Register with ModuleRegistration service using muxcore.json info
	regResp, err := client.Register(ctx, &modulev1.RegisterRequest{
		ModuleId: modInfo.ID,
		InfoJson: compactJSON,
		MeshAddr: "", // sidecar has no inbound mesh listener
	})
	if err != nil {
		log.Fatalf("registration RPC failed: %v", err)
	}
	if !regResp.Accepted {
		log.Fatalf("registration rejected: %s", regResp.Error)
	}
	log.Printf("module %q registered successfully", modInfo.ID)

	// 5. Initialise and start the module
	if err := mod.Init(ctx); err != nil {
		log.Fatalf("module init failed: %v", err)
	}
	if err := mod.Start(ctx); err != nil {
		log.Fatalf("module start failed: %v", err)
	}
	log.Printf("module %q started", modInfo.ID)

	// 6. Run until SIGTERM or SIGINT
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	sig := <-sigCh
	log.Printf("received signal %v, shutting down", sig)

	// 7. Stop the module and unregister
	if err := mod.Stop(ctx); err != nil {
		log.Printf("module stop error: %v", err)
	}

	unregResp, err := client.Unregister(ctx, &modulev1.UnregisterRequest{
		ModuleId: modInfo.ID,
	})
	if err != nil {
		log.Printf("unregister RPC failed: %v", err)
	} else if !unregResp.Acknowledged {
		log.Printf("unregister not acknowledged by core")
	} else {
		log.Printf("module %q unregistered successfully", modInfo.ID)
	}

	fmt.Println("sidecar shut down cleanly")
}
