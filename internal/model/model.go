package model

import (
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steam-webapi"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"net"
	"regexp"
	"time"
)

var (
	ErrRCON = errors.New("RCON error")
)

const (
	LenDiscordID = 18
	LenSteamID   = 17
)

// BanType defines the state of the ban for a user, 0 being no ban
type BanType int

const (
	// Unknown means the ban state could not be determined, failing-open to allowing players
	// to connect.
	Unknown BanType = -1
	// OK Ban state is clean
	OK BanType = 0
	// NoComm means the player cannot communicate while playing voice + chat
	NoComm BanType = 1
	// Banned means the player cannot join the server at all
	Banned BanType = 2
)

// BanSource defines the origin of the ban or action
type BanSource int

const (
	// System is an automatic ban triggered by the service
	System BanSource = 0
	// Bot is a ban using the discord bot interface
	Bot BanSource = 1
	// Web is a ban using the web-ui
	Web BanSource = 2
	// InGame is a ban using the sourcemod plugin
	InGame BanSource = 3
)

func (s BanSource) String() string {
	switch s {
	case System:
		return "System"
	case Bot:
		return "Bot"
	case Web:
		return "Web"
	case InGame:
		return "In-Game"
	default:
		return "Unknown"
	}
}

// Reason defined a set of predefined ban reasons
// TODO make this fully dynamic?
type Reason int

const (
	Custom           Reason = 1
	External         Reason = 2
	Cheating         Reason = 3
	Racism           Reason = 4
	Harassment       Reason = 5
	Exploiting       Reason = 6
	WarningsExceeded Reason = 7
	Spam             Reason = 8
	Language         Reason = 9
)

var reasonStr = map[Reason]string{
	Custom:           "",
	External:         "3rd party",
	Cheating:         "Cheating",
	Racism:           "Racism",
	Harassment:       "Person Harassment",
	Exploiting:       "Exploiting",
	WarningsExceeded: "Warnings Exceeding",
	Spam:             "Spam",
}

func (r Reason) String() string {
	return reasonStr[r]
}

type BanNet struct {
	NetID      int64         `db:"net_id"`
	SteamID    steamid.SID64 `db:"steam_id"`
	AuthorID   steamid.SID64 `db:"author_id"`
	CIDR       *net.IPNet    `db:"cidr"`
	Source     BanSource     `db:"source"`
	Reason     string        `db:"reason"`
	CreatedOn  time.Time     `db:"created_on" json:"created_on"`
	UpdatedOn  time.Time     `db:"updated_on" json:"updated_on"`
	ValidUntil time.Time     `db:"valid_until"`
}

func NewBan(steamID steamid.SID64, authorID steamid.SID64, duration time.Duration) *Ban {
	if duration.Seconds() == 0 {
		// 100 Years
		duration = time.Hour * 8760 * 100
	}
	return &Ban{
		SteamID:    steamID,
		AuthorID:   authorID,
		BanType:    Banned,
		Reason:     Custom,
		ReasonText: "Unspecified",
		Note:       "",
		Source:     System,
		ValidUntil: config.Now().Add(duration),
		CreatedOn:  config.Now(),
		UpdatedOn:  config.Now(),
	}
}

func NewBanNet(cidr string, reason string, duration time.Duration, source BanSource) (BanNet, error) {
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		return BanNet{}, err
	}
	if duration.Seconds() == 0 {
		// 100 Years
		duration = time.Hour * 8760 * 100
	}
	return BanNet{
		CIDR:       n,
		Source:     source,
		Reason:     reason,
		CreatedOn:  config.Now(),
		UpdatedOn:  config.Now(),
		ValidUntil: config.Now().Add(duration),
	}, nil
}

func (b BanNet) String() string {
	return fmt.Sprintf("Net: %s Origin: %s Reason: %s", b.CIDR, b.Source, b.Reason)
}

type Ban struct {
	BanID uint64 `db:"ban_id" json:"ban_id"`
	// SteamID is the steamID of the banned person
	SteamID  steamid.SID64 `db:"steam_id" json:"steam_id"`
	AuthorID steamid.SID64 `db:"author_id" json:"author_id"`
	// Reason defines the overall ban classification
	BanType BanType `db:"ban_type" json:"ban_type"`
	// Reason defines the overall ban classification
	Reason Reason `db:"reason" json:"reason"`
	// ReasonText is returned to the client when kicked trying to join the server
	ReasonText string `db:"reason_text" json:"reason_text"`
	// Note is a supplementary note added by admins that is hidden from normal view
	Note   string    `db:"note" json:"note"`
	Source BanSource `json:"ban_source" db:"ban_source"`
	// ValidUntil is when the ban will be no longer valid. 0 denotes forever
	ValidUntil time.Time `json:"valid_until" db:"valid_until"`
	CreatedOn  time.Time `db:"created_on" json:"created_on"`
	UpdatedOn  time.Time `db:"updated_on" json:"updated_on"`
}

func (b Ban) String() string {
	return fmt.Sprintf("SID: %d Origin: %s Reason: %s Type: %v",
		b.SteamID.Int64(), b.Source, b.ReasonText, b.BanType)
}

type BannedPerson struct {
	Ban                *Ban              `json:"ban"`
	Person             *Person           `json:"person"`
	HistoryChat        []logparse.SayEvt `json:"history_chat" db:"-"`
	HistoryPersonaName []string          `json:"history_personaname" db:"-"`
	HistoryConnections []string          `json:"history_connections" db:"-"`
	HistoryIP          []IPRecord        `json:"history_ip" db:"-"`
}

func NewBannedPerson() *BannedPerson {
	return &BannedPerson{
		Ban: &Ban{
			CreatedOn: config.Now(),
			UpdatedOn: config.Now(),
		},
		Person: &Person{
			CreatedOn:     config.Now(),
			UpdatedOn:     config.Now(),
			PlayerSummary: &steam_webapi.PlayerSummary{},
		},
		HistoryChat:        nil,
		HistoryPersonaName: nil,
		HistoryConnections: nil,
		HistoryIP:          nil,
	}
}

type ChatLog struct {
	Message   string
	CreatedOn time.Time
}

type IPRecord struct {
	IPAddr    net.IP    `json:"ip_addr"`
	CreatedOn time.Time `json:"created_on"`
}

// PersonIPRecord holds a composite result of the more relevant ip2location results
type PersonIPRecord struct {
	IP          net.IP
	CreatedOn   time.Time
	CityName    string
	CountryName string
	CountryCode string
	ASName      string
	ASNum       int
	ISP         string
	UsageType   string
	Threat      string
	DomainUsed  string
}

type Server struct {
	// Auto generated id
	ServerID int64 `db:"server_id" json:"server_id"`
	// ServerName is a short reference name for the server eg: us-1
	ServerName string `db:"short_name" json:"server_name"`
	// Token is the current valid authentication token that the server uses to make authenticated requests
	Token string `db:"token" json:"token"`
	// Address is the ip of the server
	Address string `db:"address" json:"address"`
	// Port is the port of the server
	Port int `db:"port" json:"port"`
	// RCON is the RCON password for the server
	RCON          string `db:"rcon" json:"-"`
	ReservedSlots int    `db:"reserved_slots" json:"reserved_slots"`
	// Password is what the server uses to generate a token to make authenticated calls
	Password string `db:"password" json:"password"`
	// TokenCreatedOn is set when changing the token
	TokenCreatedOn time.Time `db:"token_created_on" json:"token_created_on"`
	CreatedOn      time.Time `db:"created_on" json:"created_on"`
	UpdatedOn      time.Time `db:"updated_on" json:"updated_on"`
}

func (s Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.Address, s.Port)
}

func (s Server) Slots(statusSlots int) int {
	return statusSlots - s.ReservedSlots
}

func NewServer(name string, address string, port int) Server {
	return Server{
		ServerName:     name,
		Address:        address,
		Port:           port,
		RCON:           golib.RandomString(10),
		ReservedSlots:  0,
		Password:       golib.RandomString(10),
		TokenCreatedOn: time.Unix(0, 0),
		CreatedOn:      config.Now(),
		UpdatedOn:      config.Now(),
	}
}

type Person struct {
	SteamID          steamid.SID64 `db:"steam_id" json:"steam_id"`
	Name             string        `db:"name" json:"name"`
	CreatedOn        time.Time     `db:"created_on" json:"created_on"`
	UpdatedOn        time.Time     `db:"updated_on" json:"updated_on"`
	PermissionLevel  Privilege     `db:"permission_level" json:"permission_level"`
	IsNew            bool          `db:"-" json:"-"`
	DiscordID        string        `db:"discord_id" json:"discord_id"`
	IPAddr           net.IP        `db:"ip_addr" json:"ip_addr"`
	CommunityBanned  bool
	VACBans          int
	GameBans         int
	EconomyBan       string
	DaysSinceLastBan int
	*steam_webapi.PlayerSummary
}

// LoggedIn checks for a valid steamID
func (p *Person) LoggedIn() bool {
	return p.SteamID.Valid() && p.SteamID.Int64() > 0
}

// NewPerson allocates a new default person instance
func NewPerson(sid64 steamid.SID64) *Person {
	return &Person{
		SteamID:         sid64,
		IsNew:           true,
		CreatedOn:       config.Now(),
		UpdatedOn:       config.Now(),
		PlayerSummary:   &steam_webapi.PlayerSummary{},
		PermissionLevel: PAuthenticated,
	}
}

// AppealState is the current state of a users ban appeal, if any.
type AppealState int

const (
	// ASNew is a user has initiated an appeal
	ASNew AppealState = 0
	// ASDenied the appeal was denied
	ASDenied AppealState = 1
	// The appeal was granted
	//ASGranted AppealState = 2
)

type Appeal struct {
	AppealID    int         `db:"appeal_id" json:"appeal_id"`
	BanID       uint64      `db:"ban_id" json:"ban_id"`
	AppealText  string      `db:"appeal_text" json:"appeal_text"`
	AppealState AppealState `db:"appeal_state" json:"appeal_state"`
	Email       string      `db:"email" json:"email"`
	CreatedOn   time.Time   `db:"created_on" json:"created_on"`
	UpdatedOn   time.Time   `db:"updated_on" json:"updated_on"`
}

type Stats struct {
	BansTotal     int `json:"bans"`
	BansDay       int `json:"bans_day"`
	BansWeek      int `json:"bans_week"`
	BansMonth     int `json:"bans_month"`
	Bans3Month    int `json:"bans_3month"`
	Bans6Month    int `json:"bans_6month"`
	BansYear      int `json:"bans_year"`
	BansCIDRTotal int `json:"bans_cidr"`
	AppealsOpen   int `json:"appeals_open"`
	AppealsClosed int `json:"appeals_closed"`
	FilteredWords int `json:"filtered_words"`
	ServersAlive  int `json:"servers_alive"`
	ServersTotal  int `json:"servers_total"`
}

// ServerEvent is a flat struct encapsulating a parsed log event
// Fields being present is event dependent, so do not assume everything will be
// available
type ServerEvent struct {
	LogID       int64                `json:"log_id"`
	Server      *Server              `json:"server"`
	EventType   logparse.MsgType     `json:"event_type"`
	Source      *Person              `json:"source"`
	Target      *Person              `json:"target"`
	PlayerClass logparse.PlayerClass `json:"class"`
	Weapon      logparse.Weapon      `json:"weapon"`
	Damage      int                  `json:"damage"`
	Item        logparse.PickupItem  `json:"item"`
	AttackerPOS logparse.Pos         `json:"attacker_pos"`
	VictimPOS   logparse.Pos         `json:"victim_pos"`
	AssisterPOS logparse.Pos         `json:"assister_pos"`
	Extra       string               `json:"extra"`
	CreatedOn   time.Time            `json:"created_on"`
}

type Filter struct {
	WordID    int
	Word      *regexp.Regexp
	CreatedOn time.Time
}

func (f *Filter) Match(value string) bool {
	return f.Word.MatchString(value)
}

// RawLogEvent represents a full representation of a server log entry including all meta data attached to the log.
type RawLogEvent struct {
	LogID     int64             `json:"log_id"`
	Type      logparse.MsgType  `json:"event_type"`
	Event     map[string]string `json:"event"`
	Server    Server            `json:"server"`
	Player1   *Person           `json:"player1"`
	Player2   *Person           `json:"player2"`
	Assister  *Person           `json:"assister"`
	RawEvent  string            `json:"raw_event"`
	CreatedOn time.Time         `json:"created_on"`
}

// Unmarshal is just a helper to
func (e *RawLogEvent) Unmarshal(output interface{}) error {
	return logparse.Unmarshal(e.Event, output)
}

type PlayerInfo struct {
	Player  *extra.Player
	Server  *Server
	SteamID steamid.SID64
	InGame  bool
	Valid   bool
}

type FindResult struct {
	Player *extra.Player
	Server *Server
}

type LogQueryOpts struct {
	LogTypes  []logparse.MsgType `json:"log_types"`
	Limit     uint64             `json:"limit"`
	OrderDesc bool               `json:"order_desc"`
	Query     string             `json:"query"`
	SourceID  string             `json:"source_id"`
	TargetID  string             `json:"target_id"`
	Servers   []int              `json:"servers"`
}

func (lqo *LogQueryOpts) ValidRecordType(t logparse.MsgType) bool {
	if len(lqo.LogTypes) == 0 {
		// No filters == Any
		return true
	}
	for _, mt := range lqo.LogTypes {
		if mt == t {
			return true
		}
	}
	return false
}
