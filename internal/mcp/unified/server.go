package unified

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/andragon31/Ragnarok/internal/mcp"
	fenrirmcp "github.com/andragon31/Ragnarok/internal/fenrir/mcp"
	hatimcp "github.com/andragon31/Ragnarok/internal/hati/mcp"
	skollmcp "github.com/andragon31/Ragnarok/internal/skoll/mcp"
	tyrmcp "github.com/andragon31/Ragnarok/internal/tyr/mcp"

	fenrircfg "github.com/andragon31/Ragnarok/internal/fenrir/config"
	haticfg "github.com/andragon31/Ragnarok/internal/hati/config"
	skollcfg "github.com/andragon31/Ragnarok/internal/skoll/config"
	tyrcfg "github.com/andragon31/Ragnarok/internal/tyr/config"

	fenrirdb "github.com/andragon31/Ragnarok/internal/fenrir/database"
	hatidb "github.com/andragon31/Ragnarok/internal/hati/database"
	skolldb "github.com/andragon31/Ragnarok/internal/skoll/database"
	tyrdb "github.com/andragon31/Ragnarok/internal/tyr/database"
)

type Server struct {
	handlers map[string]mcp.ToolHandler
}

func NewServer(dataDir string) (*Server, error) {
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".ragnarok")
	}

	s := &Server{
		handlers: make(map[string]mcp.ToolHandler),
	}

	// 1. Initialize Fenrir
	fCfg := &fenrircfg.Config{DataDir: filepath.Join(dataDir, ".fenrir")}
	fDB, err := fenrirdb.NewDB(filepath.Join(fCfg.DataDir, "fenrir.db"))
	if err == nil {
		fenrirdb.InitSchema(fDB)
		fSrv := fenrirmcp.NewServer(fCfg, fDB)
		for k, v := range fSrv.Handlers() {
			s.handlers[k] = v
		}
	}

	// 2. Initialize Hati
	hCfg, _ := haticfg.LoadConfig(filepath.Join(dataDir, ".hati"))
	hDB, err := hatidb.NewDB(hCfg.DBPath())
	if err == nil {
		hatidb.InitSchema(hDB)
		hSrv := hatimcp.NewServer(hCfg, hDB)
		for k, v := range hSrv.Handlers() {
			s.handlers[k] = v
		}
	}

	// 3. Initialize Skoll
	sCfg, _ := skollcfg.LoadConfig(filepath.Join(dataDir, ".skoll"))
	sDB, err := skolldb.NewDB(sCfg.DBPath())
	if err == nil {
		skolldb.InitSchema(sDB)
		skSrv := skollmcp.NewServer(sCfg, sDB)
		for k, v := range skSrv.Handlers() {
			s.handlers[k] = v
		}
	}

	// 4. Initialize Tyr
	tCfg, _ := tyrcfg.LoadConfig(filepath.Join(dataDir, ".tyr"))
	tDB, err := tyrdb.NewDB(tCfg.DBPath())
	if err == nil {
		tyrdb.InitSchema(tDB)
		tSrv := tyrmcp.NewServer(tCfg, tDB)
		for k, v := range tSrv.Handlers() {
			s.handlers[k] = v
		}
	}

	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	log.Printf("Ragnarok Unified MCP server running on stdio")

	stdin := os.NewFile(uintptr(os.Stdin.Fd()), "stdin")
	stdout := os.NewFile(uintptr(os.Stdout.Fd()), "stdout")
	decoder := json.NewDecoder(stdin)
	encoder := json.NewEncoder(stdout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var raw json.RawMessage
			if err := decoder.Decode(&raw); err != nil {
				return err
			}

			var req mcp.Request
			if err := json.Unmarshal(raw, &req); err != nil {
				continue
			}

			handler, ok := s.handlers[req.Method]
			if !ok {
				// Handle standard MCP methods like list_tools, etc. if needed
				// For now, only tool calls
				continue
			}

			resp, err := handler(ctx, &req)
			if err != nil {
				continue
			}

			if err := encoder.Encode(resp); err != nil {
				return err
			}
		}
	}
}
