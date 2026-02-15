package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack/v5"
)

// ---------- helpers ----------

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// startTestServer spins up an httptest.Server with a Hub and returns
// the server, its WebSocket URL, and a cleanup func.
func startTestServer(t *testing.T) (*httptest.Server, string, func()) {
	t.Helper()

	prevIdleTimeout := SessionIdleTimeout
	SessionIdleTimeout = 150 * time.Millisecond

	// Create a temp client dir with a minimal index.html
	tmpDir := t.TempDir()
	jsDir := filepath.Join(tmpDir, "js")
	os.MkdirAll(jsDir, 0o755)
	os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte("<html>test</html>"), 0o644)
	os.WriteFile(filepath.Join(jsDir, "main.js"), []byte("// test"), 0o644)

	hub := NewHub()
	go hub.Run()

	mux := SetupRoutes(hub, tmpDir)
	srv := httptest.NewServer(mux)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

	return srv, wsURL, func() {
		SessionIdleTimeout = prevIdleTimeout
		srv.Close()
	}
}

// dialWS opens a WebSocket connection to the test server.
func dialWS(t *testing.T, wsURL string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial WS: %v", err)
	}
	return conn
}

// readEnvelope reads one JSON message from the WebSocket.
func readEnvelope(t *testing.T, conn *websocket.Conn) Envelope {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	msgType, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read WS: %v", err)
	}
	// Binary messages are msgpack-encoded GameState
	if msgType == websocket.BinaryMessage {
		var gs GameState
		if err := msgpack.Unmarshal(raw, &gs); err != nil {
			t.Fatalf("msgpack unmarshal: %v", err)
		}
		return Envelope{T: MsgState, Data: gs}
	}
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return env
}

// sendMsg sends a typed message over the WebSocket.
func sendMsg(t *testing.T, conn *websocket.Conn, msgType string, data interface{}) {
	t.Helper()
	env := Envelope{T: msgType, Data: data}
	raw, _ := json.Marshal(env)
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("write WS: %v", err)
	}
}

// dataMap extracts the Data field as map[string]interface{}.
func dataMap(t *testing.T, env Envelope) map[string]interface{} {
	t.Helper()
	raw, _ := json.Marshal(env.Data)
	var m map[string]interface{}
	json.Unmarshal(raw, &m)
	return m
}

// createAndJoin creates a session then joins it. Returns the session ID.
func createAndJoin(t *testing.T, conn *websocket.Conn, name, sname string) string {
	t.Helper()
	sendMsg(t, conn, "create", map[string]string{"name": name, "sname": sname})
	created := readEnvelope(t, conn)
	if created.T != MsgCreated {
		t.Fatalf("expected created, got %s", created.T)
	}
	sid := dataMap(t, created)["sid"].(string)

	sendMsg(t, conn, "join", map[string]string{"name": name, "sid": sid})
	joined := readEnvelope(t, conn)
	if joined.T != MsgJoined {
		t.Fatalf("expected joined, got %s", joined.T)
	}
	_ = readEnvelope(t, conn) // welcome
	return sid
}

// ---------- UUID generation tests ----------

func TestGenerateUUIDFormat(t *testing.T) {
	for i := 0; i < 20; i++ {
		id := GenerateUUID()
		if !uuidRegex.MatchString(id) {
			t.Errorf("GenerateUUID() = %q, does not match UUID v4 format", id)
		}
	}
}

func TestGenerateUUIDUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateUUID()
		if seen[id] {
			t.Fatalf("duplicate UUID generated: %s", id)
		}
		seen[id] = true
	}
}

// ---------- Session manager uses UUIDs ----------

func TestSessionIDIsUUID(t *testing.T) {
	sm := NewSessionManager()
	sess := sm.CreateSession("TestArena")
	if !uuidRegex.MatchString(sess.ID) {
		t.Errorf("session ID %q is not a valid UUID v4", sess.ID)
	}
}

// ---------- SPA routing ----------

func TestSPARoutingRoot(t *testing.T) {
	srv, _, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("GET / status = %d, want 200", resp.StatusCode)
	}
}

func TestSPARoutingUUIDPath(t *testing.T) {
	srv, _, cleanup := startTestServer(t)
	defer cleanup()

	uuid := GenerateUUID()
	resp, err := http.Get(srv.URL + "/" + uuid)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("GET /%s status = %d, want 200", uuid, resp.StatusCode)
	}
	// Should serve index.html content
	buf := make([]byte, 100)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])
	if !strings.Contains(body, "<html>") {
		t.Errorf("UUID path should serve index.html, got %q", body)
	}
}

func TestSPARoutingStaticFiles(t *testing.T) {
	srv, _, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := http.Get(srv.URL + "/js/main.js")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("GET /js/main.js status = %d, want 200", resp.StatusCode)
	}
}

func TestSPARoutingNonUUIDPath(t *testing.T) {
	srv, _, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := http.Get(srv.URL + "/not-a-uuid")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Should fall through to file server (404)
	if resp.StatusCode != 404 {
		t.Errorf("GET /not-a-uuid status = %d, want 404", resp.StatusCode)
	}
}

// ---------- Session check protocol (new code) ----------

func TestCheckSessionExists(t *testing.T) {
	srv, wsURL, cleanup := startTestServer(t)
	_ = srv
	defer cleanup()

	// Create and join a session via WS
	c1 := dialWS(t, wsURL)
	defer c1.Close()

	sid := createAndJoin(t, c1, "Pilot", "Arena")

	// Now check that session with another client
	c2 := dialWS(t, wsURL)
	defer c2.Close()

	sendMsg(t, c2, "check", map[string]string{"sid": sid})

	checked := readEnvelope(t, c2)
	if checked.T != MsgChecked {
		t.Fatalf("expected checked, got %s", checked.T)
	}
	d := dataMap(t, checked)
	if d["exists"] != true {
		t.Error("expected exists=true")
	}
	if d["sid"] != sid {
		t.Errorf("expected sid=%s, got %s", sid, d["sid"])
	}
	if d["name"] != "Arena" {
		t.Errorf("expected name=Arena, got %v", d["name"])
	}
	if d["players"].(float64) != 1 {
		t.Errorf("expected 1 player, got %v", d["players"])
	}
}

func TestCheckSessionNotExists(t *testing.T) {
	srv, wsURL, cleanup := startTestServer(t)
	_ = srv
	defer cleanup()

	c := dialWS(t, wsURL)
	defer c.Close()

	fakeSID := GenerateUUID()
	sendMsg(t, c, "check", map[string]string{"sid": fakeSID})

	checked := readEnvelope(t, c)
	if checked.T != MsgChecked {
		t.Fatalf("expected checked, got %s", checked.T)
	}
	d := dataMap(t, checked)
	if d["exists"] != false {
		t.Error("expected exists=false for non-existent session")
	}
	if d["sid"] != fakeSID {
		t.Errorf("expected sid=%s, got %v", fakeSID, d["sid"])
	}
}

// ---------- Full join-via-URL flow ----------

func TestJoinViaSessionID(t *testing.T) {
	srv, wsURL, cleanup := startTestServer(t)
	_ = srv
	defer cleanup()

	// Player 1 creates and joins a session
	c1 := dialWS(t, wsURL)
	defer c1.Close()

	sid := createAndJoin(t, c1, "Alice", "TestBattle")

	// Player 2 checks, then joins
	c2 := dialWS(t, wsURL)
	defer c2.Close()

	sendMsg(t, c2, "check", map[string]string{"sid": sid})
	checked := readEnvelope(t, c2)
	d := dataMap(t, checked)
	if d["exists"] != true {
		t.Fatal("session should exist")
	}

	sendMsg(t, c2, "join", map[string]string{"name": "Bob", "sid": sid})
	joinedMsg := readEnvelope(t, c2)
	if joinedMsg.T != MsgJoined {
		t.Fatalf("expected joined, got %s", joinedMsg.T)
	}
	joinSID := dataMap(t, joinedMsg)["sid"].(string)
	if joinSID != sid {
		t.Errorf("expected to join session %s, got %s", sid, joinSID)
	}

	welcomeMsg := readEnvelope(t, c2)
	if welcomeMsg.T != MsgWelcome {
		t.Fatalf("expected welcome, got %s", welcomeMsg.T)
	}
}

func TestJoinNonExistentSession(t *testing.T) {
	srv, wsURL, cleanup := startTestServer(t)
	_ = srv
	defer cleanup()

	c := dialWS(t, wsURL)
	defer c.Close()

	fakeSID := GenerateUUID()
	sendMsg(t, c, "join", map[string]string{"name": "Lost", "sid": fakeSID})

	errMsg := readEnvelope(t, c)
	if errMsg.T != MsgError {
		t.Fatalf("expected error, got %s", errMsg.T)
	}
}

// ---------- Session create + leave lifecycle ----------

func TestCreateAndLeaveSession(t *testing.T) {
	srv, wsURL, cleanup := startTestServer(t)
	_ = srv
	defer cleanup()

	c := dialWS(t, wsURL)
	defer c.Close()

	sid := createAndJoin(t, c, "Solo", "TempBattle")

	// Verify session exists via another client
	c2 := dialWS(t, wsURL)
	defer c2.Close()
	sendMsg(t, c2, "check", map[string]string{"sid": sid})
	checked := readEnvelope(t, c2)
	if dataMap(t, checked)["exists"] != true {
		t.Fatal("session should exist")
	}

	// Leave the session
	sendMsg(t, c, "leave", nil)

	// Give a moment for cleanup
	time.Sleep(SessionIdleTimeout + 50*time.Millisecond)

	// Session should be empty and cleaned up
	sendMsg(t, c2, "check", map[string]string{"sid": sid})
	checked2 := readEnvelope(t, c2)
	if dataMap(t, checked2)["exists"] != false {
		t.Error("session should be cleaned up after last player leaves")
	}
}

// ---------- Session list ----------

func TestListSessions(t *testing.T) {
	srv, wsURL, cleanup := startTestServer(t)
	_ = srv
	defer cleanup()

	// First, list should be empty
	c := dialWS(t, wsURL)
	defer c.Close()

	sendMsg(t, c, "list", nil)
	listMsg := readEnvelope(t, c)
	if listMsg.T != MsgSessions {
		t.Fatalf("expected sessions, got %s", listMsg.T)
	}
	// Should be empty list
	raw, _ := json.Marshal(listMsg.Data)
	var sessions []SessionInfo
	json.Unmarshal(raw, &sessions)
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}

	// Create and join a session
	c2 := dialWS(t, wsURL)
	defer c2.Close()
	createAndJoin(t, c2, "P1", "Arena1")

	// Now list should have 1 session
	sendMsg(t, c, "list", nil)
	listMsg2 := readEnvelope(t, c)
	raw2, _ := json.Marshal(listMsg2.Data)
	var sessions2 []SessionInfo
	json.Unmarshal(raw2, &sessions2)
	if len(sessions2) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions2))
	}
	if sessions2[0].Name != "Arena1" {
		t.Errorf("expected session name Arena1, got %s", sessions2[0].Name)
	}
	if sessions2[0].Players != 1 {
		t.Errorf("expected 1 player, got %d", sessions2[0].Players)
	}
}

// ---------- Game state broadcasts ----------

func TestGameStateBroadcasts(t *testing.T) {
	srv, wsURL, cleanup := startTestServer(t)
	_ = srv
	defer cleanup()

	c := dialWS(t, wsURL)
	defer c.Close()

	createAndJoin(t, c, "Tester", "StateTest")

	// Should start receiving state broadcasts
	state := readEnvelope(t, c)
	if state.T != MsgState {
		t.Fatalf("expected state broadcast, got %s", state.T)
	}
	d := dataMap(t, state)
	if d["tick"] == nil {
		t.Error("state should have tick field")
	}
	if d["p"] == nil {
		t.Error("state should have players field")
	}
	if d["pr"] == nil {
		t.Error("state should have projectiles field")
	}
}

// ---------- Input handling over WS ----------

func TestInputHandling(t *testing.T) {
	srv, wsURL, cleanup := startTestServer(t)
	_ = srv
	defer cleanup()

	c := dialWS(t, wsURL)
	defer c.Close()

	createAndJoin(t, c, "Inputter", "InputTest")

	// Send input (shouldn't error/crash)
	sendMsg(t, c, "input", ClientInput{
		MX:     500,
		MY:     500,
		Fire:   true,
		Boost:  false,
		Thresh: 100,
	})

	// Should still get state broadcasts (game didn't crash)
	env := readEnvelope(t, c)
	if env.T != MsgState {
		t.Fatalf("expected state after input, got %s", env.T)
	}
}

// ---------- Input before joining (edge case) ----------

func TestInputBeforeJoin(t *testing.T) {
	srv, wsURL, cleanup := startTestServer(t)
	_ = srv
	defer cleanup()

	c := dialWS(t, wsURL)
	defer c.Close()

	// Send input without joining - should not crash
	sendMsg(t, c, "input", ClientInput{MX: 100, MY: 100, Fire: true})

	// Connection should still work
	sendMsg(t, c, "list", nil)
	env := readEnvelope(t, c)
	if env.T != MsgSessions {
		t.Fatalf("expected sessions, got %s", env.T)
	}
}

// ---------- Multiple players in same session ----------

func TestMultiplePlayersInSession(t *testing.T) {
	srv, wsURL, cleanup := startTestServer(t)
	_ = srv
	defer cleanup()

	// Create and join session
	c1 := dialWS(t, wsURL)
	defer c1.Close()
	sid := createAndJoin(t, c1, "Alpha", "MultiTest")

	// Join with second player
	c2 := dialWS(t, wsURL)
	defer c2.Close()
	sendMsg(t, c2, "join", map[string]string{"name": "Beta", "sid": sid})
	_ = readEnvelope(t, c2) // joined
	_ = readEnvelope(t, c2) // welcome

	// Join with third player
	c3 := dialWS(t, wsURL)
	defer c3.Close()
	sendMsg(t, c3, "join", map[string]string{"name": "Gamma", "sid": sid})
	_ = readEnvelope(t, c3) // joined
	_ = readEnvelope(t, c3) // welcome

	// Check player count
	c4 := dialWS(t, wsURL)
	defer c4.Close()
	sendMsg(t, c4, "check", map[string]string{"sid": sid})
	checked := readEnvelope(t, c4)
	d := dataMap(t, checked)
	if d["players"].(float64) != 3 {
		t.Errorf("expected 3 players, got %v", d["players"])
	}
}

// ---------- Default names ----------

func TestDefaultPlayerName(t *testing.T) {
	srv, wsURL, cleanup := startTestServer(t)
	_ = srv
	defer cleanup()

	c := dialWS(t, wsURL)
	defer c.Close()

	// Create session, then join with empty name
	sendMsg(t, c, "create", map[string]string{"name": "", "sname": ""})
	created := readEnvelope(t, c)
	if created.T != MsgCreated {
		t.Fatalf("expected created, got %s", created.T)
	}
	sid := dataMap(t, created)["sid"].(string)

	sendMsg(t, c, "join", map[string]string{"name": "", "sid": sid})
	_ = readEnvelope(t, c) // joined
	welcome := readEnvelope(t, c)
	if welcome.T != MsgWelcome {
		t.Fatalf("expected welcome, got %s", welcome.T)
	}
}

// ---------- WebSocket /ws endpoint ----------

func TestWSEndpoint(t *testing.T) {
	srv, wsURL, cleanup := startTestServer(t)
	_ = srv
	defer cleanup()

	// Should be able to connect
	conn := dialWS(t, wsURL)
	defer conn.Close()

	// Should be able to send/receive
	sendMsg(t, conn, "list", nil)
	env := readEnvelope(t, conn)
	if env.T != MsgSessions {
		t.Fatalf("expected sessions response, got %s", env.T)
	}
}

// ---------- Hub client tracking ----------

func TestHubClientCount(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	if hub.ClientCount() != 0 {
		t.Errorf("expected 0 clients, got %d", hub.ClientCount())
	}
}

// ---------- Session manager tests ----------

func TestSessionManagerCreateAndGet(t *testing.T) {
	sm := NewSessionManager()
	sess := sm.CreateSession("Battle")

	got := sm.GetSession(sess.ID)
	if got == nil {
		t.Fatal("expected to find created session")
	}
	if got.Name != "Battle" {
		t.Errorf("expected name Battle, got %s", got.Name)
	}
}

func TestSessionManagerGetNonExistent(t *testing.T) {
	sm := NewSessionManager()
	got := sm.GetSession("nonexistent")
	if got != nil {
		t.Error("expected nil for non-existent session")
	}
}

func TestSessionManagerListSessions(t *testing.T) {
	sm := NewSessionManager()
	sm.CreateSession("Arena1")
	sm.CreateSession("Arena2")

	list := sm.ListSessions()
	if len(list) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(list))
	}
}

func TestSessionManagerRemovePlayer(t *testing.T) {
	prevIdleTimeout := SessionIdleTimeout
	SessionIdleTimeout = 20 * time.Millisecond
	defer func() {
		SessionIdleTimeout = prevIdleTimeout
	}()

	sm := NewSessionManager()
	sess := sm.CreateSession("TempArena")
	player := sess.Game.AddPlayer("TestPlayer")

	sm.RemovePlayer(sess.ID, player.ID)

	// Session should be cleaned up (0 players)
	time.Sleep(SessionIdleTimeout + 20*time.Millisecond)
	got := sm.GetSession(sess.ID)
	if got != nil {
		t.Error("expected session to be removed after last player leaves")
	}
}

// ---------- Util functions ----------

func TestGenerateIDLength(t *testing.T) {
	id := GenerateID(4)
	if len(id) != 8 { // 4 bytes = 8 hex chars
		t.Errorf("expected 8 chars, got %d: %s", len(id), id)
	}

	id2 := GenerateID(8)
	if len(id2) != 16 {
		t.Errorf("expected 16 chars, got %d: %s", len(id2), id2)
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		v, min, max, want float64
	}{
		{5, 0, 10, 5},
		{-1, 0, 10, 0},
		{15, 0, 10, 10},
		{0, 0, 10, 0},
		{10, 0, 10, 10},
	}
	for _, tt := range tests {
		got := Clamp(tt.v, tt.min, tt.max)
		if got != tt.want {
			t.Errorf("Clamp(%f, %f, %f) = %f, want %f", tt.v, tt.min, tt.max, got, tt.want)
		}
	}
}

func TestDistance(t *testing.T) {
	d := Distance(0, 0, 3, 4)
	if d != 5 {
		t.Errorf("Distance(0,0,3,4) = %f, want 5", d)
	}
}

func TestNormalizeAngle(t *testing.T) {
	tests := []struct {
		input, wantApprox float64
	}{
		{0, 0},
		{3.14159, 3.14159},
		{-3.14159, -3.14159},
		{7, 7 - 2*3.14159265358979},
	}
	for _, tt := range tests {
		got := NormalizeAngle(tt.input)
		diff := got - tt.wantApprox
		if diff > 0.01 || diff < -0.01 {
			t.Errorf("NormalizeAngle(%f) = %f, want ~%f", tt.input, got, tt.wantApprox)
		}
	}
}

func TestLerpAngle(t *testing.T) {
	got := LerpAngle(0, 1, 0.5)
	want := 0.5
	diff := got - want
	if diff > 0.01 || diff < -0.01 {
		t.Errorf("LerpAngle(0, 1, 0.5) = %f, want ~%f", got, want)
	}
}

// ---------- Cache-Control header ----------

func TestCacheControlHeader(t *testing.T) {
	srv, _, cleanup := startTestServer(t)
	defer cleanup()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	cc := resp.Header.Get("Cache-Control")
	if cc != "no-cache" {
		t.Errorf("expected Cache-Control: no-cache, got %q", cc)
	}
}

// ---------- Leave without joining ----------

func TestLeaveWithoutJoining(t *testing.T) {
	srv, wsURL, cleanup := startTestServer(t)
	_ = srv
	defer cleanup()

	c := dialWS(t, wsURL)
	defer c.Close()

	// Should not crash
	sendMsg(t, c, "leave", nil)

	// Should still work after
	sendMsg(t, c, "list", nil)
	env := readEnvelope(t, c)
	if env.T != MsgSessions {
		t.Fatalf("expected sessions, got %s", env.T)
	}
}

// ---------- Create session, disconnect, session cleaned up ----------

func TestDisconnectCleansUpSession(t *testing.T) {
	srv, wsURL, cleanup := startTestServer(t)
	_ = srv
	defer cleanup()

	c1 := dialWS(t, wsURL)
	sid := createAndJoin(t, c1, "Temp", "TempArena")

	// Disconnect
	c1.Close()

	// Wait for hub to process unregister
	time.Sleep(SessionIdleTimeout + 50*time.Millisecond)

	// Check if session is gone
	c2 := dialWS(t, wsURL)
	defer c2.Close()
	sendMsg(t, c2, "check", map[string]string{"sid": sid})
	checked := readEnvelope(t, c2)
	if dataMap(t, checked)["exists"] != false {
		t.Error("session should be cleaned up after disconnect")
	}
}
