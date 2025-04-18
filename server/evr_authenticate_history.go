package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/heroiclabs/nakama/v3/server/evr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	LoginStorageCollection = "Login"
	LoginHistoryStorageKey = "history"
	LoginHistoryCacheIndex = "Index_LoginHistory"
)

var (
	IgnoredLoginValues = map[string]struct{}{
		"":                {},
		"1WMHH000X00000":  {},
		"N/A":             {},
		"UNK-0":           {},
		"OVR-ORG-0":       {},
		"unknown":         {},
		"1PASH5D1P17365":  {}, // Quest Link
		"WMHD315M3010GV":  {}, // Quest link
		"VRLINKHMDQUEST":  {}, // Quest link
		"VRLINKHMDQUEST2": {}, // Quest link
		"VRLINKHMDQUEST3": {}, // Quest link
	}

	ErrPendingAuthorizationNotFound = errors.New("pending authorization not found")
)

type LoginHistoryEntry struct {
	CreatedAt time.Time         `json:"create_time"`
	UpdatedAt time.Time         `json:"update_time"`
	XPID      evr.EvrId         `json:"xpi"`
	ClientIP  string            `json:"client_ip"`
	LoginData *evr.LoginProfile `json:"login_data"`
}

func (e *LoginHistoryEntry) Key() string {
	return loginHistoryEntryKey(e.XPID, e.ClientIP)
}

func loginHistoryEntryKey(xpid evr.EvrId, clientIP string) string {
	return xpid.Token() + ":" + clientIP
}

func (e *LoginHistoryEntry) PendingCode() string {
	return fmt.Sprintf("%02d", e.CreatedAt.Nanosecond()%100)
}

func (h *LoginHistoryEntry) SystemProfile() string {
	components := []string{normalizeHeadsetType(h.LoginData.SystemInfo.HeadsetType), h.LoginData.SystemInfo.NetworkType, h.LoginData.SystemInfo.VideoCard, h.LoginData.SystemInfo.CPUModel, fmt.Sprintf("%d", h.LoginData.SystemInfo.NumPhysicalCores), fmt.Sprintf("%d", h.LoginData.SystemInfo.NumLogicalCores), fmt.Sprintf("%d", h.LoginData.SystemInfo.MemoryTotal), fmt.Sprintf("%d", h.LoginData.SystemInfo.DedicatedGPUMemory)}

	for i := range components {
		components[i] = strings.ReplaceAll(components[i], "::", ";")
	}

	return strings.Join(components, "::")
}

func (h *LoginHistoryEntry) Items() []string {
	return []string{h.ClientIP, h.LoginData.HMDSerialNumber, h.XPID.Token(), h.SystemProfile()}
}

func (h *LoginHistoryEntry) ItemMap() map[string]struct{} {
	return map[string]struct{}{
		h.ClientIP:                  {},
		h.LoginData.HMDSerialNumber: {},
		h.XPID.Token():              {},
		h.SystemProfile():           {}}
}

var _ = IndexedVersionedStorable(&LoginHistory{})

type LoginHistory struct {
	History                  map[string]*LoginHistoryEntry      `json:"history"`                    // map[deviceID]DeviceHistoryEntry
	Cache                    []string                           `json:"cache"`                      // list of IP addresses, EvrID's, HMD Serial Numbers, and System Data
	XPIs                     map[string]time.Time               `json:"xpis"`                       // list of XPIs
	ClientIPs                map[string]time.Time               `json:"client_ips"`                 // map[clientIP]time.Time
	AuthorizedIPs            map[string]time.Time               `json:"authorized_client_ips"`      // map[clientIP]time.Time
	DeniedClientAddresses    []string                           `json:"denied_client_addrs"`        // list of denied IPs
	PendingAuthorizations    map[string]*LoginHistoryEntry      `json:"pending_authorizations"`     // map[XPID:ClientIP]LoginHistoryEntry
	SecondDegreeAlternates   []string                           `json:"second_degree"`              // []userID
	AlternateMap             map[string][]*AlternateSearchMatch `json:"alternate_accounts"`         // map of alternate user IDs and what they have in common
	GroupNotifications       map[string]map[string]time.Time    `json:"notified_groups"`            // list of groups that have been notified of this alternate login
	IgnoreDisabledAlternates bool                               `json:"ignore_disabled_alternates"` // Ignore disabled alternates
	userID                   string                             // user ID
	version                  string                             // storage record version
}

func (h *LoginHistory) StorageMeta() StorageMeta {
	version := "*"
	if h != nil && h.version != "" {
		version = h.version
	}
	return StorageMeta{
		Collection:      LoginStorageCollection,
		Key:             LoginHistoryStorageKey,
		PermissionRead:  runtime.STORAGE_PERMISSION_NO_READ,
		PermissionWrite: runtime.STORAGE_PERMISSION_NO_WRITE,
		Version:         version,
	}
}

func (LoginHistory) StorageIndex() *StorageIndexMeta {
	return &StorageIndexMeta{
		Name:           LoginHistoryCacheIndex,
		Collection:     LoginStorageCollection,
		Key:            LoginHistoryStorageKey,
		Fields:         []string{"cache", "xpis", "client_ips", "second_degree", "alternate_matches", "denied_client_addrs"},
		SortableFields: nil,
		MaxEntries:     10000000,
		IndexOnly:      false,
	}
}

func (h *LoginHistory) SetStorageVersion(userID, version string) {
	h.userID = userID
	h.version = version
}

func NewLoginHistory(userID string) *LoginHistory {
	return &LoginHistory{
		History:                make(map[string]*LoginHistoryEntry),
		Cache:                  make([]string, 0),
		XPIs:                   make(map[string]time.Time),
		ClientIPs:              make(map[string]time.Time),
		AuthorizedIPs:          make(map[string]time.Time),
		DeniedClientAddresses:  make([]string, 0),
		PendingAuthorizations:  make(map[string]*LoginHistoryEntry),
		SecondDegreeAlternates: make([]string, 0),
		AlternateMap:           make(map[string][]*AlternateSearchMatch),
		GroupNotifications:     make(map[string]map[string]time.Time),
		userID:                 userID,
		version:                "*", // don't overwrite existing data
	}
}

func (h *LoginHistory) AlternateIDs() (firstDegree []string, secondDegree []string) {
	var (
		firstIDs  = make([]string, 0, len(h.AlternateMap))
		secondIDs = make([]string, 0, len(h.SecondDegreeAlternates))
	)

	for userID := range h.AlternateMap {
		firstIDs = append(firstIDs, userID)
	}

	for _, userID := range h.SecondDegreeAlternates {
		if _, found := h.AlternateMap[userID]; !found {
			secondIDs = append(secondIDs, userID)
		}
	}

	slices.Sort(firstIDs)
	slices.Sort(secondIDs)

	return slices.Compact(firstIDs), slices.Compact(secondIDs)
}

func (h *LoginHistory) Update(xpid evr.EvrId, clientIP string, loginData *evr.LoginProfile) {
	if e := h.Get(xpid, clientIP); e != nil {
		e.UpdatedAt = time.Now().UTC()
		e.LoginData = loginData
		return
	} else {
		h.Insert(&LoginHistoryEntry{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			XPID:      xpid,
			ClientIP:  clientIP,
			LoginData: loginData,
		})
	}
}

func (h *LoginHistory) Get(xpid evr.EvrId, clientIP string) *LoginHistoryEntry {
	if h.History == nil {
		return nil
	}
	key := loginHistoryEntryKey(xpid, clientIP)
	if e, found := h.History[key]; found {
		return e
	}
	return nil
}

func (h *LoginHistory) Insert(entry *LoginHistoryEntry) {
	if h.History == nil {
		h.History = make(map[string]*LoginHistoryEntry)
	}
	h.History[entry.Key()] = entry
}

func (h *LoginHistory) AuthorizeIPWithCode(ip, code string) error {
	for _, e := range h.PendingAuthorizations {
		if e.ClientIP == ip {

			if e.PendingCode() != code {
				return fmt.Errorf("invalid code %s for IP %s", code, ip)
			}

			_ = h.AuthorizeIP(ip)
			return nil
		}
	}

	return ErrPendingAuthorizationNotFound
}

func (h *LoginHistory) AuthorizeIP(ip string) bool {
	if h.AuthorizedIPs == nil {
		h.AuthorizedIPs = make(map[string]time.Time)
	}
	h.RemovePendingAuthorizationIP(ip)
	_, found := h.AuthorizedIPs[ip]
	h.AuthorizedIPs[ip] = time.Now().UTC()

	return !found
}

func (h *LoginHistory) IsAuthorizedIP(ip string) (isAuthorized bool) {
	if h.AuthorizedIPs != nil {
		if _, found := h.AuthorizedIPs[ip]; found {
			return true
		}
	}
	return false
}

func (h *LoginHistory) AddPendingAuthorizationIP(xpid evr.EvrId, clientIP string, loginData *evr.LoginProfile) *LoginHistoryEntry {
	if h.PendingAuthorizations == nil {
		h.PendingAuthorizations = make(map[string]*LoginHistoryEntry)
	}
	e := &LoginHistoryEntry{
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		XPID:      xpid,
		ClientIP:  clientIP,
		LoginData: loginData,
	}

	h.PendingAuthorizations[e.Key()] = e
	return e
}

func (h *LoginHistory) GetPendingAuthorizationIP(ip string) *LoginHistoryEntry {
	if h.PendingAuthorizations == nil {
		return nil
	}
	return h.PendingAuthorizations[ip]
}

func (h *LoginHistory) RemovePendingAuthorizationIP(ip string) {
	if h.PendingAuthorizations == nil {
		return
	}
	delete(h.PendingAuthorizations, ip)
}

func (h *LoginHistory) NotifyGroup(groupID string, threshold time.Time) bool {

	userIDs := make([]string, 0, len(h.AlternateMap)+len(h.SecondDegreeAlternates))
	for k := range h.AlternateMap {
		userIDs = append(userIDs, k)
	}

	if len(h.SecondDegreeAlternates) > 0 {
		userIDs = append(userIDs, h.SecondDegreeAlternates...)
	}

	if len(userIDs) == 0 {
		return false
	}

	slices.Sort(userIDs)
	userIDs = slices.Compact(userIDs)

	if h.GroupNotifications == nil {
		h.GroupNotifications = make(map[string]map[string]time.Time)

	}
	if _, found := h.GroupNotifications[groupID]; !found {
		h.GroupNotifications[groupID] = make(map[string]time.Time)
	}

	updated := false
	// Check if the group has already been notified for all of the userIDs within the threshold
	for _, userID := range userIDs {
		if t, found := h.GroupNotifications[groupID][userID]; !found || t.Before(threshold) {
			h.GroupNotifications[groupID][userID] = time.Now()
			updated = true
		}
	}

	return updated
}

func (h *LoginHistory) UpdateAlternates(ctx context.Context, nk runtime.NakamaModule, excludeUserIDs ...string) (hasDisabledAlts bool, err error) {
	matches, err := LoginAlternateSearch(ctx, nk, h)
	if err != nil {
		return false, fmt.Errorf("error searching for alternate logins: %w", err)
	}

	h.AlternateMap = make(map[string][]*AlternateSearchMatch, len(matches))
	h.SecondDegreeAlternates = make([]string, 0, len(matches))

	userIDs := make([]string, 0, len(matches))
	for _, m := range matches {
		userIDs = append(userIDs, m.otherHistory.userID)
	}
	slices.Sort(userIDs)
	userIDs = slices.Compact(userIDs)

	// Remove excluded user IDs
	for i := 0; i < len(userIDs); i++ {
		if slices.Contains(excludeUserIDs, userIDs[i]) {
			userIDs = slices.Delete(userIDs, i, i+1)
			i--
		}
	}

	if accounts, err := nk.AccountsGetId(ctx, userIDs); err != nil {
		return false, fmt.Errorf("error getting accounts for user IDs %v: %w", userIDs, err)
	} else {
		for _, a := range accounts {
			if !h.IgnoreDisabledAlternates && a.GetDisableTime() != nil && !a.GetDisableTime().AsTime().IsZero() {
				hasDisabledAlts = true
			}
		}
	}

	secondMap := make(map[string]struct{}, len(matches))
	for _, m := range matches {

		h.AlternateMap[m.otherHistory.userID] = append(h.AlternateMap[m.otherHistory.userID], m)
		// add second-level alternates
		for id := range m.otherHistory.AlternateMap {
			secondMap[id] = struct{}{}
		}
	}

	// Filter the second-degree alternates
	h.SecondDegreeAlternates = make([]string, 0, len(secondMap))

	for userID := range h.AlternateMap {

		if userID == h.userID {
			continue
		}

		if _, found := h.AlternateMap[userID]; found {
			continue
		}

		h.SecondDegreeAlternates = append(h.SecondDegreeAlternates, userID)
	}

	slices.Sort(h.SecondDegreeAlternates)

	// Remove duplicates
	h.SecondDegreeAlternates = slices.Compact(h.SecondDegreeAlternates)

	// Check if the alternates have changed

	return hasDisabledAlts, nil
}

func (h *LoginHistory) rebuildCache() {
	h.Cache = make([]string, 0, len(h.History)*4)
	h.XPIs = make(map[string]time.Time, len(h.History))
	h.ClientIPs = make(map[string]time.Time, len(h.History))

	// Rebuild the cache from each history entry
	for _, e := range h.History {

		h.Cache = append(h.Cache, e.XPID.Token(), e.ClientIP, e.LoginData.HMDSerialNumber, e.SystemProfile())

		if !e.XPID.IsNil() {
			evrIDStr := e.XPID.String()
			if t, found := h.XPIs[evrIDStr]; !found || t.Before(e.UpdatedAt) {
				h.XPIs[evrIDStr] = e.UpdatedAt
			}
		}

		if e.ClientIP != "" {
			if t, found := h.ClientIPs[e.ClientIP]; !found || t.Before(e.UpdatedAt) {
				h.ClientIPs[e.ClientIP] = e.UpdatedAt
			}
		}
	}
	if h.DeniedClientAddresses == nil {
		h.DeniedClientAddresses = make([]string, 0)
	}
	h.Cache = append(h.Cache, h.DeniedClientAddresses...)

	// Sort and compact the cache
	slices.Sort(h.Cache)
	h.Cache = slices.Compact(h.Cache)

	// Remove ignored values
	for i := 0; i < len(h.Cache); i++ {
		if _, ok := IgnoredLoginValues[h.Cache[i]]; ok {
			h.Cache = append(h.Cache[:i], h.Cache[i+1:]...)
			i--
		}
	}
}

func (h *LoginHistory) MarshalJSON() ([]byte, error) {

	if h.userID == "" {
		return nil, fmt.Errorf("missing user ID")
	}

	// Clear authorized IPs that haven't been used in over 30 days
	for ip := range h.AuthorizedIPs {
		lastUse, found := h.ClientIPs[ip]
		if !found || time.Since(lastUse) > 30*24*time.Hour {
			delete(h.AuthorizedIPs, ip)
		}
	}

	// Alias to avoid recursion during marshaling
	type Alias LoginHistory
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(h),
	}

	// Keep the history size under 5MB
	var bytes []byte
	var err error
	for {

		// Rebuild the cache
		h.rebuildCache()

		bytes, err = json.Marshal(aux)
		if err != nil {
			return nil, fmt.Errorf("error marshalling display name history: %w", err)
		}

		if len(bytes) < 5*1024*1024 {
			return bytes, nil
		}

		// Remove the oldest entries
		for range 5 {
			oldest := time.Now()
			oldestKey := ""
			for k, e := range h.History {
				if e.UpdatedAt.Before(oldest) {
					oldest = e.UpdatedAt
					oldestKey = k
				}
			}
			delete(h.History, oldestKey)
			h.rebuildCache()
		}
	}
}

func LoginHistoryRegexSearch(ctx context.Context, nk runtime.NakamaModule, pattern string, limit int) (map[string]*LoginHistory, error) {

	query := fmt.Sprintf("+value.cache:/%s/", pattern)
	// Perform the storage list operation

	cursor := ""
	histories := make(map[string]*LoginHistory)
	for {
		result, cursor, err := nk.StorageIndexList(ctx, SystemUserID, LoginHistoryCacheIndex, query, limit, nil, cursor)
		if err != nil {
			return nil, fmt.Errorf("error listing display name history: %w", err)
		}

		for _, obj := range result.Objects {
			var history LoginHistory
			if err := json.Unmarshal([]byte(obj.Value), &history); err != nil {
				return nil, fmt.Errorf("error unmarshalling display name history: %w", err)
			}
			history.userID = obj.UserId
			history.version = obj.Version
			histories[obj.UserId] = &history
		}

		if cursor == "" {
			break
		}
	}
	return histories, nil
}

func AccountGetDeviceID(ctx context.Context, db *sql.DB, nk runtime.NakamaModule, deviceID string) (*api.Account, error) {
	found := true

	// Look for an existing account.
	query := "SELECT user_id FROM user_device WHERE id = $1"
	var dbUserID string
	err := db.QueryRowContext(ctx, query, deviceID).Scan(&dbUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			found = false
		} else {
			return nil, status.Error(codes.Internal, "Error finding user account by device id.")
		}
	}

	if found {
		return nk.AccountGetId(ctx, dbUserID)
	}

	return nil, status.Error(codes.NotFound, "User account not found.")
}
