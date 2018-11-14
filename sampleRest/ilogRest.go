package myserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-martini/martini"
	"github.com/jbsturgeon/ilogger"

	util "github.com/Novetta/go.util"
	"github.com/Novetta/security/kerbtypes"
)

// const block for reading env variables
const (
	DataDirEnv      = "CEREBRO_DATA_DIR"
	EnableUILogging = "CEREBRO_ENABLE_UI_LOGGING"
)

var (
	iLogEnabled bool
	iLogger     = &ilogger.ILog{}
)

func init() {
	// Set data directory
	myDataDir := util.GetSetEnv(DataDirEnv, "/opt/myapp/data")
	myDataDir = strings.TrimRight(myDataDir, "/")

	if strings.Compare(strings.ToLower(util.GetSetEnv(EnableUILogging, "false")), "true") == 0 {
		myUIDir := filepath.Join(myDataDir, "uilogs")
		iLogger.NewFile(myUIDir, -1, -1)

		iLogEnabled = true
	}
	log.Printf("UI logging is enabled: %v.", iLogEnabled)
}

func registerILogEndpoints(h http.Handler) {
	m := h.(*martini.ClassicMartini)
	m.Post("/uilog", writeILogMessageHandler)
}

func writeILogMessageHandler(w http.ResponseWriter, r *http.Request, user *kerbtypes.User) {
	if iLogEnabled {
		var m string
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		msg := fmt.Sprintf("User (%s), Session (%s): %s", user.GetID(), user.SessionID, m)
		iLogger.Log(ilogger.LMandatory, msg)
	}
}
