package controller

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
)

// MysekaiController handles mysekai module build/render requests.
type MysekaiController struct {
	drawing *service.DrawingService
	userData *service.UserDataService
	masterData *service.MasterDataService
}

func NewMysekaiController(drawing *service.DrawingService, userData *service.UserDataService, masterData *service.MasterDataService) *MysekaiController {
	return &MysekaiController{drawing: drawing, userData: userData, masterData: masterData}
}

func (c *MysekaiController) ensure() error {
	if c == nil || c.drawing == nil {
		return fmt.Errorf("mysekai controller is not initialized")
	}
	return nil
}

func (c *MysekaiController) Build(req interface{}) (interface{}, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("mysekai request is empty")
	}
	return req, nil
}

func (c *MysekaiController) RenderResource(req interface{}) ([]byte, error) {
	payload, err := c.BuildResourceRequest(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateMysekaiResource(payload)
}

func (c *MysekaiController) BuildResourceRequest(req interface{}) (model.MysekaiResourceRequest, error) {
	if err := c.ensure(); err != nil {
		return model.MysekaiResourceRequest{}, err
	}

	region := "jp"
	if typed, ok := req.(model.MysekaiResourceRequest); ok {
		return typed, nil
	}

	if payload, ok := req.(map[string]interface{}); ok {
		if rawRegion, ok := payload["region"].(string); ok && rawRegion != "" {
			region = rawRegion
		}
		if _, hasProfile := payload["profile"]; hasProfile {
			var built model.MysekaiResourceRequest
			buf, err := json.Marshal(payload)
			if err != nil {
				return model.MysekaiResourceRequest{}, err
			}
			if err := json.Unmarshal(buf, &built); err != nil {
				return model.MysekaiResourceRequest{}, err
			}
			return built, nil
		}
	}

	if c.userData == nil {
		return model.MysekaiResourceRequest{}, fmt.Errorf("mysekai resource requires user data")
	}

	raw, err := c.userData.RawBytes()
	if err != nil {
		return model.MysekaiResourceRequest{}, err
	}

	var merged map[string]interface{}
	if err := json.Unmarshal(raw, &merged); err != nil {
		return model.MysekaiResourceRequest{}, err
	}

	profile := c.mysekaiProfileCard(region, merged)
	if profile == nil {
		return model.MysekaiResourceRequest{}, fmt.Errorf("mysekai resource requires profile data")
	}

	gateID, gateLevel := extractMysekaiGate(merged)
	phenoms := extractMysekaiPhenoms(merged)
	visitCharacters := c.extractVisitCharacters(merged)
	siteResources := c.extractSiteResourceNumbers(merged)

	return model.MysekaiResourceRequest{
		Profile:             profile,
		Phenoms:             phenoms,
		GateID:              gateID,
		GateLevel:           gateLevel,
		GateIconPath:        fmt.Sprintf("mysekai/gate_icon/gate_%d.png", gateID),
		VisitCharacters:     visitCharacters,
		SiteResourceNumbers: siteResources,
	}, nil
}

func (c *MysekaiController) extractVisitCharacters(merged map[string]interface{}) []model.MysekaiVisitCharacter {
	visit, ok := merged["userMysekaiGateCharacterVisit"].(map[string]interface{})
	if !ok {
		return []model.MysekaiVisitCharacter{}
	}
	groupMap := c.loadMysekaiGameCharacterUnitGroups()
	characters, ok := visit["userMysekaiGateCharacters"].([]interface{})
	if !ok {
		return []model.MysekaiVisitCharacter{}
	}
	result := make([]model.MysekaiVisitCharacter, 0, len(characters))
	seen := map[int]struct{}{}
	for _, item := range characters {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		groupID := intNumber(entry["mysekaiGameCharacterUnitGroupId"], 0)
		group := groupMap[groupID]
		if _, hasSecond := group["gameCharacterUnitId2"]; hasSecond {
			continue
		}
		displayUnitID := intNumber(group["gameCharacterUnitId1"], 0)
		if displayUnitID == 0 {
			continue
		}
		if _, ok := seen[displayUnitID]; ok {
			continue
		}
		seen[displayUnitID] = struct{}{}
		gameCharacterID := c.gameCharacterIDByUnitID(displayUnitID)
		var memoriaPath *string
		if gameCharacterID > 0 {
			memoriaPath = stringPtr(fmt.Sprintf("mysekai/item_preview/material/item_memoria_%d.png", gameCharacterID))
		}
		var reservationPath *string
		if boolValue(entry["isReservation"]) {
			reservationPath = stringPtr("mysekai/invitationcard.png")
		}
		sdPath := fmt.Sprintf("character/character_sd_l/chr_sp_%d.png", displayUnitID)
		result = append(result, model.MysekaiVisitCharacter{
			SDImagePath:         sdPath,
			MemoriaImagePath:    memoriaPath,
			IsRead:              false,
			IsReservation:       boolValue(entry["isReservation"]),
			ReservationIconPath: reservationPath,
		})
		if len(result) >= 6 {
			break
		}
	}
	return result
}

func (c *MysekaiController) extractSiteResourceNumbers(merged map[string]interface{}) []model.MysekaiSiteResourceNumber {
	updated := nestedList(merged, "userMysekaiHarvestMaps")
	if len(updated) == 0 {
		return []model.MysekaiSiteResourceNumber{}
	}
	counts := map[int]map[string]int{5: {}, 7: {}, 6: {}, 8: {}}
	for _, item := range updated {
		siteMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		siteID := intNumber(siteMap["mysekaiSiteId"], 0)
		if _, ok := counts[siteID]; !ok {
			counts[siteID] = map[string]int{}
		}
		drops, _ := siteMap["userMysekaiSiteHarvestResourceDrops"].([]interface{})
		for _, rawDrop := range drops {
			drop, ok := rawDrop.(map[string]interface{})
			if !ok {
				continue
			}
			if status, _ := drop["mysekaiSiteHarvestResourceDropStatus"].(string); status != "before_drop" {
				continue
			}
			key := fmt.Sprintf("%s_%d", drop["resourceType"], intNumber(drop["resourceId"], 0))
			counts[siteID][key] += intNumber(drop["quantity"], 0)
		}
	}
	order := []int{5, 7, 6, 8}
	materialMap := c.loadIconNameMap("mysekaiMaterials.json", "iconAssetbundleName")
	materialRarityMap := c.loadFieldMap("mysekaiMaterials.json", "mysekaiMaterialRarityType")
	itemMap := c.loadIconNameMap("mysekaiItems.json", "iconAssetbundleName")
	musicRecordMap := c.loadMusicRecordJacketMap()
	result := make([]model.MysekaiSiteResourceNumber, 0, len(order))
	for _, siteID := range order {
		resMap := counts[siteID]
		keys := make([]string, 0, len(resMap))
		for key := range resMap {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			ri := resourceRarity(keys[i], materialRarityMap)
			rj := resourceRarity(keys[j], materialRarityMap)
			if ri != rj {
				return ri > rj
			}
			if resMap[keys[i]] != resMap[keys[j]] {
				return resMap[keys[i]] > resMap[keys[j]]
			}
			return keys[i] < keys[j]
		})
		resources := make([]model.MysekaiResourceNumber, 0, len(keys))
		for _, key := range keys {
			imgPath, hasRecord := c.resourceImagePath(key, materialMap, itemMap, musicRecordMap, merged)
			if imgPath == "" {
				continue
			}
			resources = append(resources, model.MysekaiResourceNumber{
				ImagePath:           imgPath,
				Number:              resMap[key],
				TextColor:           resourceTextColor(key, materialRarityMap),
				HasMusicRecord:      hasRecord,
				MusicRecordIconPath: musicRecordIconPath(hasRecord),
			})
		}
		if len(resources) == 0 {
			continue
		}
		result = append(result, model.MysekaiSiteResourceNumber{
			ImagePath:       fmt.Sprintf("mysekai/site/sitemap/texture/img_harvest_site_%d.png", siteID),
			ResourceNumbers: resources,
		})
	}
	return result
}

func (c *MysekaiController) resourceImagePath(key string, materialMap, itemMap, musicRecordMap map[int]string, merged map[string]interface{}) (string, bool) {
	parts := strings.Split(key, "_")
	if len(parts) < 2 {
		return "", false
	}
	id := intNumber(parts[len(parts)-1], 0)
	typeKey := strings.TrimSuffix(key, fmt.Sprintf("_%d", id))
	switch typeKey {
	case "mysekai_material":
		if icon := materialMap[id]; icon != "" {
			return fmt.Sprintf("mysekai/thumbnail/material/%s.png", icon), false
		}
	case "material":
		return fmt.Sprintf("thumbnail/material_rip/material%d.png", id), false
	case "mysekai_item":
		if icon := itemMap[id]; icon != "" {
			return fmt.Sprintf("mysekai/thumbnail/item/%s.png", icon), false
		}
	case "mysekai_music_record":
		if jacket := musicRecordMap[id]; jacket != "" {
			return fmt.Sprintf("music/jacket/%s/%s.png", jacket, jacket), hasMysekaiMusicRecord(merged, id)
		}
	}
	return "", false
}

func hasMysekaiMusicRecord(merged map[string]interface{}, recordID int) bool {
	items := nestedList(merged, "userMysekaiMusicRecords")
	for _, item := range items {
		entry, ok := item.(map[string]interface{})
		if ok && intNumber(entry["mysekaiMusicRecordId"], 0) == recordID {
			return true
		}
	}
	return false
}

func nestedList(root map[string]interface{}, key string) []interface{} {
	if items, ok := root[key].([]interface{}); ok {
		return items
	}
	if updated, ok := root["updatedResources"].(map[string]interface{}); ok {
		if items, ok := updated[key].([]interface{}); ok {
			return items
		}
	}
	return nil
}

func musicRecordIconPath(hasRecord bool) string {
	if hasRecord {
		return "mysekai/music_record.png"
	}
	return ""
}

func resourceTextColor(key string, materialRarityMap map[int]string) []int {
	rarity := resourceRarity(key, materialRarityMap)
	if rarity >= 2 {
		return []int{200, 50, 0}
	}
	if rarity == 1 {
		return []int{50, 0, 200}
	}
	return []int{100, 100, 100}
}

func resourceRarity(key string, materialRarityMap map[int]string) int {
	mostRare := map[string]struct{}{
		"mysekai_material_5":  {},
		"mysekai_material_12": {},
		"mysekai_material_20": {},
		"mysekai_material_24": {},
		"mysekai_fixture_121": {},
		"material_17":         {},
		"material_170":        {},
	}
	rare := map[string]struct{}{
		"mysekai_material_32": {},
		"mysekai_material_33": {},
		"mysekai_material_34": {},
		"mysekai_material_61": {},
		"mysekai_material_64": {},
		"mysekai_material_65": {},
		"mysekai_material_66": {},
	}
	if _, ok := mostRare[key]; ok {
		return 2
	}
	if _, ok := rare[key]; ok {
		return 1
	}
	if strings.HasPrefix(key, "mysekai_music_record") {
		return 1
	}
	if strings.HasPrefix(key, "mysekai_material_") {
		parts := strings.Split(key, "_")
		if len(parts) > 0 {
			id := intNumber(parts[len(parts)-1], 0)
			switch materialRarityMap[id] {
			case "rarity_3":
				return 2
			case "rarity_2":
				return 1
			}
		}
	}
	return 0
}

func (c *MysekaiController) loadMysekaiGameCharacterUnitGroups() map[int]map[string]interface{} {
	return c.loadMasterDataMapByID("mysekaiGameCharacterUnitGroups.json")
}

func (c *MysekaiController) loadIconNameMap(filename, field string) map[int]string {
	items := c.loadMasterDataMapByID(filename)
	result := make(map[int]string, len(items))
	for id, item := range items {
		if value, ok := item[field].(string); ok {
			result[id] = value
		}
	}
	return result
}

func (c *MysekaiController) loadFieldMap(filename, field string) map[int]string {
	items := c.loadMasterDataMapByID(filename)
	result := make(map[int]string, len(items))
	for id, item := range items {
		if value, ok := item[field].(string); ok {
			result[id] = value
		}
	}
	return result
}

func (c *MysekaiController) loadMusicRecordJacketMap() map[int]string {
	records := c.loadMasterDataMapByID("mysekaiMusicRecords.json")
	musics := c.loadMasterDataMapByID("musics.json")
	result := make(map[int]string, len(records))
	for id, record := range records {
		externalID := intNumber(record["externalId"], 0)
		if externalID == 0 {
			continue
		}
		if music, ok := musics[externalID]; ok {
			if assetbundle, ok := music["assetbundleName"].(string); ok {
				result[id] = assetbundle
			}
		}
	}
	return result
}

func (c *MysekaiController) loadMasterDataMapByID(filename string) map[int]map[string]interface{} {
	result := map[int]map[string]interface{}{}
	if c.masterData == nil {
		return result
	}
	path := filepath.Join(c.masterData.GetDataDir(), filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return result
	}
	var items []map[string]interface{}
	if err := json.Unmarshal(data, &items); err != nil {
		return result
	}
	for _, item := range items {
		id := intNumber(item["id"], 0)
		if id != 0 {
			result[id] = item
		}
	}
	return result
}

func (c *MysekaiController) loadMasterDataList(filename string) []map[string]interface{} {
	if c.masterData == nil {
		return nil
	}
	path := filepath.Join(c.masterData.GetDataDir(), filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var items []map[string]interface{}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil
	}
	return items
}

func (c *MysekaiController) gameCharacterIDByUnitID(unitID int) int {
	items := c.loadMasterDataMapByID("gameCharacterUnits.json")
	if item, ok := items[unitID]; ok {
		return intNumber(item["gameCharacterId"], 0)
	}
	return 0
}

func extractMysekaiGate(merged map[string]interface{}) (int, int) {
	visit, ok := merged["userMysekaiGateCharacterVisit"].(map[string]interface{})
	if !ok {
		return 1, 1
	}
	gate, ok := visit["userMysekaiGate"].(map[string]interface{})
	if !ok {
		return 1, 1
	}
	gateID := intNumber(gate["mysekaiGateId"], 1)
	gateLevel := intNumber(gate["mysekaiGateLevel"], 1)
	if gateID <= 0 {
		gateID = 1
	}
	if gateLevel <= 0 {
		gateLevel = 1
	}
	return gateID, gateLevel
}

func extractMysekaiPhenoms(merged map[string]interface{}) []model.MysekaiPhenomRequest {
	rawSchedules, ok := merged["mysekaiPhenomenaSchedules"].([]interface{})
	if !ok {
		return []model.MysekaiPhenomRequest{}
	}

	nowMs := int64Number(merged["now"], 0)
	if updated, ok := merged["updatedResources"].(map[string]interface{}); ok {
		if v := int64Number(updated["now"], 0); v > 0 {
			nowMs = v
		}
	}
	now := time.Now()
	if nowMs > 0 {
		now = time.UnixMilli(nowMs)
	}
	const hour1 = 4
	const hour2 = 16
	phenomStart := time.Date(now.Year(), now.Month(), now.Day(), hour1, 0, 0, 0, now.Location())
	if now.Hour() < hour1 {
		phenomStart = phenomStart.Add(-24 * time.Hour)
	}
	currentIdx := 1
	if now.Hour() >= hour1 && now.Hour() < hour2 {
		currentIdx = 0
	}

	phenoms := make([]model.MysekaiPhenomRequest, 0, len(rawSchedules))
	for i, item := range rawSchedules {
		schedule, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		phenomID := intNumber(schedule["mysekaiPhenomenaId"], 1)
		text := phenomStart.Add(time.Duration(i) * 12 * time.Hour).Format("15:04")
		bg := []int{255, 255, 255, 75}
		fg := []int{125, 125, 125, 255}
		if i == currentIdx {
			bg = []int{255, 255, 255, 150}
			fg = []int{0, 0, 0, 255}
		}
		phenoms = append(phenoms, model.MysekaiPhenomRequest{
			RefreshReason:  "natural",
			ImagePath:      fmt.Sprintf("mysekai/thumbnail/phenomena/%s.png", mysekaiPhenomIconName(phenomID)),
			BackgroundFill: bg,
			Text:           text,
			TextFill:       fg,
		})
	}
	return phenoms
}

func mysekaiPhenomIconName(phenomID int) string {
	icons := map[int]string{
		1:  "env_sunny",
		2:  "env_evening",
		3:  "env_night",
		4:  "env_fine",
		5:  "env_fullmoon",
		6:  "env_rain",
		7:  "env_rainnight",
		8:  "env_cloud",
		9:  "env_thunder",
		10: "env_snow",
		11: "env_snownight",
		12: "env_rainbow",
		13: "env_universe",
		14: "env_meteorshower",
		15: "env_sekai",
	}
	if icon, ok := icons[phenomID]; ok {
		return icon
	}
	return "env_default"
}

func intNumber(value interface{}, fallback int) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
		return fallback
	default:
		return fallback
	}
}

func int64Number(value interface{}, fallback int64) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	default:
		return fallback
	}
}

func boolValue(value interface{}) bool {
	if v, ok := value.(bool); ok {
		return v
	}
	return false
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func assetExists(masterData *service.MasterDataService, rel string) bool {
	if masterData == nil {
		return false
	}
	base := filepath.Clean(`Z:/pjskdata/Data`)
	if rel == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(base, filepath.FromSlash(rel)))
	return err == nil
}

func stringValue(value interface{}) string {
	if v, ok := value.(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func mergedMapFromUserData(userData *service.UserDataService) map[string]interface{} {
	if userData == nil {
		return map[string]interface{}{}
	}
	raw, err := userData.RawBytes()
	if err != nil {
		return map[string]interface{}{}
	}
	var merged map[string]interface{}
	if err := json.Unmarshal(raw, &merged); err != nil {
		return map[string]interface{}{}
	}
	return merged
}

func (c *MysekaiController) mysekaiProfileCard(region string, merged map[string]interface{}) *model.ProfileCardRequest {
	if c.userData == nil {
		return nil
	}
	profile := c.userData.ProfileCard(region)
	if profile == nil {
		return nil
	}
	if updated, ok := merged["userMysekaiGamedata"].(map[string]interface{}); ok {
		if level := intNumber(updated["mysekaiRank"], 0); level > 0 {
			profile.MysekaiLevel = &level
		}
	}
	return profile
}

func (c *MysekaiController) obtainedMysekaiFixtureIDs(merged map[string]interface{}, blueprints map[int]map[string]interface{}) map[int]struct{} {
	result := map[int]struct{}{}
	for _, raw := range nestedList(merged, "userMysekaiBlueprints") {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		blueprintID := intNumber(item["mysekaiBlueprintId"], 0)
		blueprint, ok := blueprints[blueprintID]
		if !ok {
			continue
		}
		if stringValue(blueprint["mysekaiCraftType"]) != "mysekai_fixture" {
			continue
		}
		targetID := intNumber(blueprint["craftTargetId"], 0)
		if targetID != 0 {
			result[targetID] = struct{}{}
		}
	}
	return result
}

func birthdayCharacterID(characters map[int]map[string]interface{}, fixtureName string) int {
	for id, item := range characters {
		givenName := stringValue(item["givenName"])
		if givenName != "" && strings.HasSuffix(fixtureName, "（"+givenName+"）") {
			return id
		}
	}
	return 0
}

func fixtureThumbnailPath(item map[string]interface{}) string {
	assetbundleName := stringValue(item["assetbundleName"])
	if assetbundleName == "" {
		return ""
	}
	if stringValue(item["mysekaiFixtureType"]) == "surface_appearance" {
		layoutType := stringValue(item["mysekaiSettableLayoutType"])
		if layoutType == "" {
			layoutType = "floor_appearance"
		}
		return fmt.Sprintf("mysekai/thumbnail/surface_appearance/%s/tex_%s_%s_1.png", assetbundleName, assetbundleName, layoutType)
	}
	return fmt.Sprintf("mysekai/thumbnail/fixture/%s_1.png", assetbundleName)
}

func fixtureColorImages(item map[string]interface{}) []model.MysekaiFixtureColorImage {
	base := fixtureThumbnailPath(item)
	if base == "" {
		return nil
	}
	images := []model.MysekaiFixtureColorImage{{ImagePath: base}}
	if colors, ok := item["mysekaiFixtureAnotherColors"].([]interface{}); ok {
		for index, raw := range colors {
			color, _ := raw.(map[string]interface{})
			colorCode := stringValue(color["colorCode"])
			assetbundleName := stringValue(item["assetbundleName"])
			if assetbundleName == "" {
				continue
			}
			path := fmt.Sprintf("mysekai/thumbnail/fixture/%s_%d.png", assetbundleName, index+2)
			if stringValue(item["mysekaiFixtureType"]) == "surface_appearance" {
				layoutType := stringValue(item["mysekaiSettableLayoutType"])
				if layoutType == "" {
					layoutType = "floor_appearance"
				}
				path = fmt.Sprintf("mysekai/thumbnail/surface_appearance/%s/tex_%s_%s_%d.png", assetbundleName, assetbundleName, layoutType, index+2)
			}
			var codePtr *string
			if colorCode != "" {
				codePtr = &colorCode
			}
			images = append(images, model.MysekaiFixtureColorImage{ImagePath: path, ColorCode: codePtr})
		}
	}
	return images
}

func fixtureBasicInfo(item map[string]interface{}) []string {
	boolLabel := func(ok bool, yes, no string) string {
		if ok {
			return yes
		}
		return no
	}
	info := []string{
		boolLabel(boolValue(item["isAssembled"]), "【🔨可制作】", "【❌不可制作】"),
		boolLabel(boolValue(item["isDisassembled"]), "【♻️可回收】", "【❌不可回收】"),
	}
	playerAction := stringValue(item["mysekaiFixturePlayerActionType"]) != "" && stringValue(item["mysekaiFixturePlayerActionType"]) != "no_action"
	info = append(info, boolLabel(playerAction, "【👋玩家可交互】", "【❌玩家不可交互】"))
	info = append(info, boolLabel(boolValue(item["isGameCharacterAction"]), "【🎡角色可交互】", "【❌角色无交互】"))
	return info
}

func fixtureBlueprintInfo(blueprint map[string]interface{}) []string {
	boolLabel := func(ok bool, yes, no string) string {
		if ok {
			return yes
		}
		return no
	}
	var info []string
	info = append(info, boolLabel(boolValue(blueprint["isEnableSketch"]), "【📝蓝图可抄写】", "【蓝图不可抄写】"))
	info = append(info, boolLabel(boolValue(blueprint["isObtainedByConvert"]), "【🎁蓝图可合成】", "【蓝图不可合成】"))
	limit := intNumber(blueprint["craftCountLimit"], 0)
	if limit > 0 {
		info = append(info, fmt.Sprintf("【最多制作%d次】", limit))
	} else {
		info = append(info, "【无制作次数限制】")
	}
	return info
}

func fixtureTags(item map[string]interface{}, tags map[int]map[string]interface{}) []string {
	group, ok := item["mysekaiFixtureTagGroup"].(map[string]interface{})
	if !ok {
		return nil
	}
	var result []string
	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("mysekaiFixtureTagId%d", i)
		if id := intNumber(group[key], 0); id != 0 {
			if tag, ok := tags[id]; ok {
				name := stringValue(tag["name"])
				if name != "" {
					result = append(result, name)
				}
			}
		}
	}
	return result
}

func findFixtureBlueprint(items []map[string]interface{}, fixtureID int) map[string]interface{} {
	for _, item := range items {
		if stringValue(item["mysekaiCraftType"]) == "mysekai_fixture" && intNumber(item["craftTargetId"], 0) == fixtureID {
			return item
		}
	}
	return nil
}

func fixtureCostMaterials(blueprintID int, costs []map[string]interface{}, c *MysekaiController) []model.MysekaiFixtureMaterial {
	var result []model.MysekaiFixtureMaterial
	for _, item := range costs {
		if intNumber(item["mysekaiBlueprintId"], 0) != blueprintID {
			continue
		}
		mid := intNumber(item["mysekaiMaterialId"], 0)
		if mid == 0 {
			continue
		}
		imgPath, _ := c.resourceImagePath(fmt.Sprintf("mysekai_material_%d", mid), c.loadIconNameMap("mysekaiMaterials.json", "iconAssetbundleName"), nil, nil, nil)
		if imgPath == "" {
			continue
		}
		result = append(result, model.MysekaiFixtureMaterial{
			ImagePath: imgPath,
			Quantity:  intNumber(item["quantity"], 0),
		})
	}
	return result
}

func fixtureRecycleMaterials(fixtureID int, items []map[string]interface{}, c *MysekaiController) []model.MysekaiFixtureMaterial {
	var result []model.MysekaiFixtureMaterial
	for _, item := range items {
		if intNumber(item["mysekaiFixtureId"], 0) != fixtureID {
			continue
		}
		mid := intNumber(item["mysekaiMaterialId"], 0)
		if mid == 0 {
			continue
		}
		imgPath, _ := c.resourceImagePath(fmt.Sprintf("mysekai_material_%d", mid), c.loadIconNameMap("mysekaiMaterials.json", "iconAssetbundleName"), nil, nil, nil)
		if imgPath == "" {
			continue
		}
		result = append(result, model.MysekaiFixtureMaterial{
			ImagePath: imgPath,
			Quantity:  intNumber(item["quantity"], 0),
		})
	}
	return result
}

func (c *MysekaiController) fixtureReactionCharacterGroups(fixtureID int) []model.MysekaiReactionCharacterGroups {
	path := filepath.Clean(`Z:/pjskdata/Data/mysekai/system/fixture_reaction_data/fixture_reaction_data.json`)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil
	}
	rawItems, _ := parsed["FixturerRactions"].([]interface{})
	grouped := map[int][][]int{}
	for _, raw := range rawItems {
		item, ok := raw.(map[string]interface{})
		if !ok || intNumber(item["FixtureId"], 0) != fixtureID {
			continue
		}
		reactions, _ := item["ReactionCharacter"].([]interface{})
		for _, rr := range reactions {
			entry, ok := rr.(map[string]interface{})
			if !ok {
				continue
			}
			chars, _ := entry["CharacterUnitIds"].([]interface{})
			var ids []int
			var paths []string
			for _, ch := range chars {
				cuid := intNumber(ch, 0)
				if cuid == 0 {
					continue
				}
				ids = append(ids, cuid)
				paths = append(paths, fmt.Sprintf("chara_icon/%s.png", charaIconName(cuid)))
			}
			if len(ids) == 0 {
				continue
			}
			grouped[len(ids)] = append(grouped[len(ids)], ids)
		}
	}
	var result []model.MysekaiReactionCharacterGroups
	counts := make([]int, 0, len(grouped))
	for n := range grouped {
		counts = append(counts, n)
	}
	sort.Ints(counts)
	for _, n := range counts {
		var pathGroups [][]string
		for _, ids := range grouped[n] {
			var paths []string
			for _, cuid := range ids {
				paths = append(paths, fmt.Sprintf("chara_icon/%s.png", charaIconName(cuid)))
			}
			pathGroups = append(pathGroups, paths)
		}
		result = append(result, model.MysekaiReactionCharacterGroups{
			Number:                n,
			CharacterUnitIDGroups: grouped[n],
			CharaIconPathGroups:   pathGroups,
		})
	}
	return result
}

func charaIconName(cuid int) string {
	names := map[int]string{
		1: "ick", 2: "saki", 3: "hnm", 4: "shiho", 5: "mnr", 6: "hrk", 7: "airi", 8: "szk",
		9: "khn", 10: "an", 11: "akt", 12: "toya", 13: "tks", 14: "emu", 15: "nene", 16: "rui",
		17: "knd", 18: "mfy", 19: "ena", 20: "mzk", 21: "miku", 22: "rin", 23: "len", 24: "luka",
		25: "meiko", 26: "kaito", 27: "miku_light_sound", 28: "miku_idol", 29: "miku_street",
		30: "miku_theme_park", 31: "miku_school_refusal", 32: "rin", 33: "rin", 34: "rin", 35: "rin",
		36: "rin", 37: "len", 38: "len", 39: "len", 40: "len", 41: "len", 42: "luka", 43: "luka",
		44: "luka", 45: "luka", 46: "luka", 47: "meiko", 48: "meiko", 49: "meiko", 50: "meiko",
		51: "meiko", 52: "kaito", 53: "kaito", 54: "kaito", 55: "kaito", 56: "kaito",
	}
	if name, ok := names[cuid]; ok {
		return name
	}
	return "miku"
}

func formatMysekaiQuantity(quantity int) string {
	if quantity >= 10000 {
		return fmt.Sprintf("%dk", quantity/1000)
	}
	if quantity >= 1000 {
		return fmt.Sprintf("%dk%d", quantity/1000, (quantity%1000)/100)
	}
	return strconv.Itoa(quantity)
}

func nestedInt(root map[string]interface{}, parent, child string) int {
	m, ok := root[parent].(map[string]interface{})
	if !ok {
		return 0
	}
	return intNumber(m[child], 0)
}

func parseIntTokens(query string) []int {
	fields := strings.FieldsFunc(query, func(r rune) bool {
		return r == ',' || r == ' ' || r == '，' || r == '\t' || r == '\n'
	})
	var ids []int
	seen := map[int]struct{}{}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		id, err := strconv.Atoi(field)
		if err != nil || id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func (c *MysekaiController) RenderFixtureList(req interface{}) ([]byte, error) {
	payload, err := c.BuildFixtureListRequest(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateMysekaiFixtureList(payload)
}

func (c *MysekaiController) BuildFixtureListRequest(req interface{}) (model.MysekaiFixtureListRequest, error) {
	if err := c.ensure(); err != nil {
		return model.MysekaiFixtureListRequest{}, err
	}

	region := "jp"
	showID := true
	if typed, ok := req.(model.MysekaiFixtureListRequest); ok {
		return typed, nil
	}
	if payload, ok := req.(map[string]interface{}); ok {
		if rawRegion, ok := payload["region"].(string); ok && rawRegion != "" {
			region = rawRegion
		}
		if rawShowID, exists := payload["show_id"]; exists {
			showID = boolValue(rawShowID)
		}
		if _, hasMainGenres := payload["main_genres"]; hasMainGenres {
			var built model.MysekaiFixtureListRequest
			buf, err := json.Marshal(payload)
			if err != nil {
				return model.MysekaiFixtureListRequest{}, err
			}
			if err := json.Unmarshal(buf, &built); err != nil {
				return model.MysekaiFixtureListRequest{}, err
			}
			return built, nil
		}
	}

	fixturesData := c.loadMasterDataList("mysekaiFixtures.json")
	mainGenreMap := c.loadMasterDataMapByID("mysekaiFixtureMainGenres.json")
	subGenreMap := c.loadMasterDataMapByID("mysekaiFixtureSubGenres.json")
	blueprints := c.loadMasterDataMapByID("mysekaiBlueprints.json")
	characters := c.loadMasterDataMapByID("gameCharacters.json")

	obtainedFixtureIDs := c.obtainedMysekaiFixtureIDs(mergedMapFromUserData(c.userData), blueprints)

	type fixtureRow struct {
		fixture  model.MysekaiFixture
		genreID  int
		subID    int
		obtained bool
	}
	grouped := map[int]map[int][]fixtureRow{}
	mainProgressAll := map[int]int{}
	mainProgressObtained := map[int]int{}
	subProgressAll := map[int]map[int]int{}
	subProgressObtained := map[int]map[int]int{}
	totalAll := 0
	totalObtained := 0

	for _, item := range fixturesData {
		fid := intNumber(item["id"], 0)
		if fid == 0 {
			continue
		}
		if strings.EqualFold(stringValue(item["mysekaiFixtureType"]), "gate") {
			continue
		}
		mainGenreID := intNumber(item["mysekaiFixtureMainGenreId"], -1)
		subGenreID := intNumber(item["mysekaiFixtureSubGenreId"], -1)
		if fid == 4 {
			subGenreID = 14
		}
		if _, ok := map[int]struct{}{4: {}, 5: {}, 7: {}, 8: {}, 9: {}, 10: {}, 11: {}, 12: {}, 13: {}}[mainGenreID]; ok {
			subGenreID = -1
		}

		if _, ok := grouped[mainGenreID]; !ok {
			grouped[mainGenreID] = map[int][]fixtureRow{}
			subProgressAll[mainGenreID] = map[int]int{}
			subProgressObtained[mainGenreID] = map[int]int{}
		}

		obtained := false
		if len(obtainedFixtureIDs) == 0 {
			obtained = true
		} else {
			_, obtained = obtainedFixtureIDs[fid]
		}

		var charID *int
		if cid := birthdayCharacterID(characters, stringValue(item["name"])); cid != 0 {
			charID = &cid
		}

		row := fixtureRow{
			fixture: model.MysekaiFixture{
				ID:          fid,
				ImagePath:   fixtureThumbnailPath(item),
				CharacterID: charID,
				Obtained:    obtained,
			},
			genreID:  mainGenreID,
			subID:    subGenreID,
			obtained: obtained,
		}
		grouped[mainGenreID][subGenreID] = append(grouped[mainGenreID][subGenreID], row)

		if charID == nil {
			totalAll++
			if obtained {
				totalObtained++
			}
			mainProgressAll[mainGenreID]++
			if obtained {
				mainProgressObtained[mainGenreID]++
			}
			subProgressAll[mainGenreID][subGenreID]++
			if obtained {
				subProgressObtained[mainGenreID][subGenreID]++
			}
		}
	}

	mainGenreIDs := make([]int, 0, len(grouped))
	for genreID := range grouped {
		mainGenreIDs = append(mainGenreIDs, genreID)
	}
	sort.Ints(mainGenreIDs)

	mainGenres := make([]model.MysekaiFixtureMainGenre, 0, len(mainGenreIDs))
	for _, genreID := range mainGenreIDs {
		subGenreIDs := make([]int, 0, len(grouped[genreID]))
		for subID := range grouped[genreID] {
			subGenreIDs = append(subGenreIDs, subID)
		}
		sort.Ints(subGenreIDs)

		subGenres := make([]model.MysekaiFixtureSubGenre, 0, len(subGenreIDs))
		for _, subID := range subGenreIDs {
			rows := grouped[genreID][subID]
			if len(rows) == 0 {
				continue
			}
			fixtures := make([]model.MysekaiFixture, 0, len(rows))
			for _, row := range rows {
				fixtures = append(fixtures, row.fixture)
			}
			subGenre := model.MysekaiFixtureSubGenre{
				Fixtures: fixtures,
			}
			if subID != -1 && len(grouped[genreID]) > 1 {
				if info, ok := subGenreMap[subID]; ok {
					name := stringValue(info["name"])
					imagePath := fmt.Sprintf("mysekai/icon/category_icon/%s.png", stringValue(info["assetbundleName"]))
					subGenre.Name = &name
					subGenre.ImagePath = &imagePath
					if total := subProgressAll[genreID][subID]; total > 0 {
						msg := fmt.Sprintf("%d/%d (%.1f%%)", subProgressObtained[genreID][subID], total, float64(subProgressObtained[genreID][subID])*100/float64(total))
						subGenre.ProgressMessage = &msg
					}
				}
			}
			subGenres = append(subGenres, subGenre)
		}
		if len(subGenres) == 0 {
			continue
		}
		info := mainGenreMap[genreID]
		mainGenre := model.MysekaiFixtureMainGenre{
			Name:      stringValue(info["name"]),
			ImagePath: fmt.Sprintf("mysekai/icon/category_icon/%s.png", stringValue(info["assetbundleName"])),
			SubGenres: subGenres,
		}
		if total := mainProgressAll[genreID]; total > 0 {
			msg := fmt.Sprintf("%d/%d (%.1f%%)", mainProgressObtained[genreID], total, float64(mainProgressObtained[genreID])*100/float64(total))
			mainGenre.ProgressMessage = &msg
		}
		mainGenres = append(mainGenres, mainGenre)
	}

	reqOut := model.MysekaiFixtureListRequest{
		ShowID:     showID,
		MainGenres: mainGenres,
	}
	reqOut.Profile = c.mysekaiProfileCard(region, mergedMapFromUserData(c.userData))
	if totalAll > 0 {
		msg := fmt.Sprintf("总收集进度（不含生日家具）: %d/%d (%.1f%%)", totalObtained, totalAll, float64(totalObtained)*100/float64(totalAll))
		reqOut.ProgressMessage = &msg
	}
	return reqOut, nil
}

func (c *MysekaiController) RenderFixtureDetail(req interface{}) ([]byte, error) {
	payload, err := c.BuildFixtureDetailRequest(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateMysekaiFixtureDetail(payload)
}

func (c *MysekaiController) BuildFixtureDetailRequest(req interface{}) ([]model.MysekaiFixtureDetailRequest, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}

	if typed, ok := req.([]model.MysekaiFixtureDetailRequest); ok {
		return typed, nil
	}
	if payload, ok := req.(map[string]interface{}); ok {
		if _, hasTitle := payload["title"]; hasTitle {
			var built []model.MysekaiFixtureDetailRequest
			buf, err := json.Marshal([]interface{}{payload})
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(buf, &built); err != nil {
				return nil, err
			}
			return built, nil
		}
	}

	query := ""
	if payload, ok := req.(map[string]interface{}); ok {
		query = strings.TrimSpace(stringValue(payload["query"]))
	}
	if query == "" {
		return nil, fmt.Errorf("mysekai fixture detail requires fixture id query")
	}

	fixtureIDs := parseIntTokens(query)
	if len(fixtureIDs) == 0 {
		return nil, fmt.Errorf("mysekai fixture detail invalid query: %s", query)
	}

	fixtureMap := c.loadMasterDataMapByID("mysekaiFixtures.json")
	mainGenreMap := c.loadMasterDataMapByID("mysekaiFixtureMainGenres.json")
	subGenreMap := c.loadMasterDataMapByID("mysekaiFixtureSubGenres.json")
	blueprints := c.loadMasterDataList("mysekaiBlueprints.json")
	blueprintCosts := c.loadMasterDataList("mysekaiBlueprintMysekaiMaterialCosts.json")
	onlyDisassemble := c.loadMasterDataList("mysekaiFixtureOnlyDisassembleMaterials.json")
	tags := c.loadMasterDataMapByID("mysekaiFixtureTags.json")

	var requests []model.MysekaiFixtureDetailRequest
	for _, fid := range fixtureIDs {
		fixture, ok := fixtureMap[fid]
		if !ok {
			continue
		}
		mainGenreID := intNumber(fixture["mysekaiFixtureMainGenreId"], 0)
		subGenreID := intNumber(fixture["mysekaiFixtureSubGenreId"], 0)
		mainGenre := mainGenreMap[mainGenreID]
		subGenre := subGenreMap[subGenreID]

		title := fmt.Sprintf("【JP-%d】%s", fid, stringValue(fixture["name"]))
		images := fixtureColorImages(fixture)
		reqItem := model.MysekaiFixtureDetailRequest{
			Title:              title,
			Images:             images,
			MainGenreName:      stringValue(mainGenre["name"]),
			MainGenreImagePath: fmt.Sprintf("mysekai/icon/category_icon/%s.png", stringValue(mainGenre["assetbundleName"])),
			Size: map[string]int{
				"width":  nestedInt(fixture, "gridSize", "width"),
				"depth":  nestedInt(fixture, "gridSize", "depth"),
				"height": nestedInt(fixture, "gridSize", "height"),
			},
			FirstPutCost:  intNumber(fixture["firstPutCost"], 0),
			SecondPutCost: intNumber(fixture["secondPutCost"], 0),
			BasicInfo:     fixtureBasicInfo(fixture),
			Tags:          fixtureTags(fixture, tags),
		}
		if subGenreID != 0 {
			subName := stringValue(subGenre["name"])
			subPath := fmt.Sprintf("mysekai/icon/category_icon/%s.png", stringValue(subGenre["assetbundleName"]))
			reqItem.SubGenreName = &subName
			reqItem.SubGenreImagePath = &subPath
		}
		if bp := findFixtureBlueprint(blueprints, fid); bp != nil {
			reqItem.BasicInfo = append(reqItem.BasicInfo, fixtureBlueprintInfo(bp)...)
			reqItem.CostMaterials = fixtureCostMaterials(intNumber(bp["id"], 0), blueprintCosts, c)
		}
		reqItem.RecycleMaterials = fixtureRecycleMaterials(fid, onlyDisassemble, c)
		reqItem.ReactionCharacterGroups = c.fixtureReactionCharacterGroups(fid)
		requests = append(requests, reqItem)
	}
	if len(requests) == 0 {
		return nil, fmt.Errorf("mysekai fixture detail found no valid fixtures")
	}
	return requests, nil
}

func (c *MysekaiController) RenderDoorUpgrade(req interface{}) ([]byte, error) {
	payload, err := c.BuildDoorUpgradeRequest(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateMysekaiDoorUpgrade(payload)
}

func (c *MysekaiController) BuildDoorUpgradeRequest(req interface{}) (model.MysekaiDoorUpgradeRequest, error) {
	if err := c.ensure(); err != nil {
		return model.MysekaiDoorUpgradeRequest{}, err
	}
	region := "jp"
	specGateID := 0
	if typed, ok := req.(model.MysekaiDoorUpgradeRequest); ok {
		return typed, nil
	}
	if payload, ok := req.(map[string]interface{}); ok {
		if rawRegion, ok := payload["region"].(string); ok && rawRegion != "" {
			region = rawRegion
		}
		if rawQuery, ok := payload["query"].(string); ok {
			ids := parseIntTokens(rawQuery)
			if len(ids) > 0 {
				specGateID = ids[0]
			}
		}
		if _, hasGate := payload["gate_materials"]; hasGate {
			var built model.MysekaiDoorUpgradeRequest
			buf, err := json.Marshal(payload)
			if err != nil {
				return model.MysekaiDoorUpgradeRequest{}, err
			}
			if err := json.Unmarshal(buf, &built); err != nil {
				return model.MysekaiDoorUpgradeRequest{}, err
			}
			return built, nil
		}
	}

	merged := mergedMapFromUserData(c.userData)
	userMaterials := map[int]int{}
	for _, raw := range nestedList(merged, "userMysekaiMaterials") {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		userMaterials[intNumber(item["mysekaiMaterialId"], 0)] = intNumber(item["quantity"], 0)
	}
	specLevels := map[int]int{}
	for _, raw := range nestedList(merged, "userMysekaiGates") {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		gid := intNumber(item["mysekaiGateId"], 0)
		if gid != 0 {
			specLevels[gid] = intNumber(item["mysekaiGateLevel"], 0)
		}
	}

	const gateMaxLevel = 40
	type tempItem struct {
		Mid         int
		Quantity    int
		Color       []int
		SumQuantity string
	}
	gateTemp := map[int][][]tempItem{}
	for _, item := range c.loadMasterDataList("mysekaiGateMaterialGroups.json") {
		groupID := intNumber(item["groupId"], 0)
		if groupID == 0 {
			continue
		}
		gid := groupID / 1000
		level := groupID % 1000
		if gid == 0 || level <= 0 || level > gateMaxLevel {
			continue
		}
		if _, ok := gateTemp[gid]; !ok {
			gateTemp[gid] = make([][]tempItem, gateMaxLevel)
		}
		gateTemp[gid][level-1] = append(gateTemp[gid][level-1], tempItem{
			Mid:         intNumber(item["mysekaiMaterialId"], 0),
			Quantity:    intNumber(item["quantity"], 0),
			Color:       []int{50, 50, 50},
			SumQuantity: "",
		})
	}

	if specGateID == 0 {
		for gid, level := range specLevels {
			if level != gateMaxLevel && level > specLevels[specGateID] {
				specGateID = gid
			}
		}
	}
	if specGateID != 0 {
		if level, ok := specLevels[specGateID]; ok && level == gateMaxLevel {
			return model.MysekaiDoorUpgradeRequest{}, fmt.Errorf("queried gate already max level")
		}
		if mats, ok := gateTemp[specGateID]; ok {
			gateTemp = map[int][][]tempItem{specGateID: mats}
		}
	}

	green := []int{0, 200, 0}
	red := []int{200, 0, 0}
	materialIcons := c.loadIconNameMap("mysekaiMaterials.json", "iconAssetbundleName")

	var gateMaterials []model.MysekaiGateMaterials
	gateIDs := make([]int, 0, len(gateTemp))
	for gid := range gateTemp {
		gateIDs = append(gateIDs, gid)
	}
	sort.Ints(gateIDs)
	for _, gid := range gateIDs {
		levelMats := gateTemp[gid]
		specLevel := specLevels[gid]
		if specLevel > 0 && specLevel < len(levelMats) {
			levelMats = levelMats[specLevel:]
		}
		sumMaterials := map[int]int{}
		var outLevels []model.MysekaiGateLevelMaterials
		for idx, items := range levelMats {
			if len(items) == 0 {
				continue
			}
			levelNo := idx + 1 + specLevel
			levelColor := []int{50, 50, 50}
			var outItems []model.MysekaiGateMaterialItem
			for _, item := range items {
				sumMaterials[item.Mid] += item.Quantity
				userQty := userMaterials[item.Mid]
				color := green
				if userQty < sumMaterials[item.Mid] {
					color = red
					levelColor = red
				}
				outItems = append(outItems, model.MysekaiGateMaterialItem{
					ImagePath:   fmt.Sprintf("mysekai/thumbnail/material/%s.png", materialIcons[item.Mid]),
					Quantity:    item.Quantity,
					Color:       color,
					SumQuantity: fmt.Sprintf("%s/%d", formatMysekaiQuantity(userQty), sumMaterials[item.Mid]),
				})
			}
			outLevels = append(outLevels, model.MysekaiGateLevelMaterials{
				Level: levelNo,
				Color: levelColor,
				Items: outItems,
			})
		}
		levelCopy := specLevel
		gateMaterials = append(gateMaterials, model.MysekaiGateMaterials{
			ID:             gid,
			Level:          &levelCopy,
			LevelMaterials: outLevels,
		})
	}

	reqOut := model.MysekaiDoorUpgradeRequest{
		GateMaterials: gateMaterials,
	}
	reqOut.Profile = c.mysekaiProfileCard(region, merged)
	return reqOut, nil
}

func (c *MysekaiController) RenderMusicRecord(req interface{}) ([]byte, error) {
	payload, err := c.BuildMusicRecordRequest(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateMysekaiMusicRecord(payload)
}

func (c *MysekaiController) BuildMusicRecordRequest(req interface{}) (model.MysekaiMusicrecordRequest, error) {
	if err := c.ensure(); err != nil {
		return model.MysekaiMusicrecordRequest{}, err
	}
	region := "jp"
	showID := false
	if typed, ok := req.(model.MysekaiMusicrecordRequest); ok {
		return typed, nil
	}
	if payload, ok := req.(map[string]interface{}); ok {
		if rawRegion, ok := payload["region"].(string); ok && rawRegion != "" {
			region = rawRegion
		}
		if rawShowID, exists := payload["show_id"]; exists {
			showID = boolValue(rawShowID)
		}
		if _, hasCategory := payload["category_musicrecords"]; hasCategory {
			var built model.MysekaiMusicrecordRequest
			buf, err := json.Marshal(payload)
			if err != nil {
				return model.MysekaiMusicrecordRequest{}, err
			}
			if err := json.Unmarshal(buf, &built); err != nil {
				return model.MysekaiMusicrecordRequest{}, err
			}
			return built, nil
		}
	}

	merged := mergedMapFromUserData(c.userData)
	obtainedRecords := map[int]int64{}
	for _, raw := range nestedList(merged, "userMysekaiMusicRecords") {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		obtainedRecords[intNumber(item["mysekaiMusicRecordId"], 0)] = int64Number(item["obtainedAt"], 0)
	}

	records := c.loadMasterDataList("mysekaiMusicRecords.json")
	musicTags := c.loadMasterDataList("musicTags.json")
	musics := c.loadMasterDataMapByID("musics.json")
	limitedTimes := c.loadMasterDataList("limitedTimeMusics.json")
	categoryMids := map[string][]int{
		"light_music_club": {},
		"idol":             {},
		"street":           {},
		"theme_park":       {},
		"school_refusal":   {},
		"vocaloid":         {},
		"other":            {},
	}
	midObtainedAt := map[int]int64{}
	recordIDByMusicID := map[int]int{}
	limitedByMusic := map[int][]map[string]interface{}{}
	for _, item := range limitedTimes {
		mid := intNumber(item["musicId"], 0)
		if mid != 0 {
			limitedByMusic[mid] = append(limitedByMusic[mid], item)
		}
	}

	tagByMusicID := map[int]string{}
	for _, item := range musicTags {
		mid := intNumber(item["musicId"], 0)
		tag := stringValue(item["musicTag"])
		if mid == 0 || tag == "" || tag == "all" || tag == "vocaloid" {
			continue
		}
		if _, exists := tagByMusicID[mid]; !exists {
			tagByMusicID[mid] = tag
		}
	}

	for _, record := range records {
		if stringValue(record["mysekaiMusicTrackType"]) != "music" {
			continue
		}
		rid := intNumber(record["id"], 0)
		mid := intNumber(record["externalId"], 0)
		if rid == 0 || mid == 0 {
			continue
		}
		if mid == 241 || mid == 290 {
			continue
		}
		music := musics[mid]
		if music == nil {
			continue
		}
		nowMs := time.Now().UnixMilli()
		if int64Number(music["publishedAt"], 0) > nowMs {
			continue
		}
		if windows := limitedByMusic[mid]; len(windows) > 0 && !isMusicAvailableNow(windows, nowMs) {
			continue
		}
		recordIDByMusicID[mid] = rid
		if ts, ok := obtainedRecords[rid]; ok {
			midObtainedAt[mid] = ts
		}
		tag := tagByMusicID[mid]
		if tag == "" {
			tag = "vocaloid"
		}
		categoryMids[tag] = append(categoryMids[tag], mid)
	}

	unitTagMap := map[string]string{
		"light_music_club": "icon_light_sound.png",
		"idol":             "icon_idol.png",
		"street":           "icon_street.png",
		"theme_park":       "icon_theme_park.png",
		"school_refusal":   "icon_school_refusal.png",
		"vocaloid":         "icon_piapro.png",
		"other":            "",
	}

	totalNum := 0
	obtainedNum := 0
	var categories []model.MysekaiCategoryMusicrecord
	order := []string{"light_music_club", "street", "idol", "theme_park", "school_refusal", "vocaloid", "other"}
	for _, tag := range order {
		mids := categoryMids[tag]
		sort.Slice(mids, func(i, j int) bool {
			left, lok := midObtainedAt[mids[i]]
			right, rok := midObtainedAt[mids[j]]
			if lok && rok {
				return left < right
			}
			if lok != rok {
				return lok
			}
			return mids[i] < mids[j]
		})
		categoryTotal := len(mids)
		categoryObtained := 0
		var musicrecords []model.MysekaiMusicrecord
		for _, mid := range mids {
			totalNum++
			if _, ok := midObtainedAt[mid]; ok {
				obtainedNum++
				categoryObtained++
			}
			music := musics[mid]
			assetbundle := stringValue(music["assetbundleName"])
			if assetbundle == "" {
				continue
			}
			rec := model.MysekaiMusicrecord{
				ImagePath: fmt.Sprintf("music/jacket/%s/%s.png", assetbundle, assetbundle),
				Obtained:  midObtainedAt[mid] != 0,
			}
			if showID {
				midCopy := mid
				rec.ID = &midCopy
			}
			musicrecords = append(musicrecords, rec)
		}
		if categoryTotal == 0 {
			continue
		}
		msg := fmt.Sprintf("%d/%d (%.1f%%)", categoryObtained, categoryTotal, float64(categoryObtained)*100/float64(categoryTotal))
		categories = append(categories, model.MysekaiCategoryMusicrecord{
			Tag:             tag,
			TagIconPath:     unitTagMap[tag],
			ProgressMessage: &msg,
			Musicrecords:    musicrecords,
		})
	}

	reqOut := model.MysekaiMusicrecordRequest{
		CategoryMusicrecords: categories,
	}
	reqOut.Profile = c.mysekaiProfileCard(region, merged)
	if totalNum > 0 {
		msg := fmt.Sprintf("总收集进度: %d/%d (%.1f%%)", obtainedNum, totalNum, float64(obtainedNum)*100/float64(totalNum))
		reqOut.ProgressMessage = &msg
	}
	return reqOut, nil
}

func (c *MysekaiController) RenderTalkList(req interface{}) ([]byte, error) {
	payload, err := c.BuildTalkListRequest(req)
	if err != nil {
		return nil, err
	}
	return c.drawing.GenerateMysekaiTalkList(payload)
}

func (c *MysekaiController) BuildTalkListRequest(req interface{}) (model.MysekaiTalkListRequest, error) {
	if err := c.ensure(); err != nil {
		return model.MysekaiTalkListRequest{}, err
	}
	region := "jp"
	query := ""
	if typed, ok := req.(model.MysekaiTalkListRequest); ok {
		return typed, nil
	}
	if payload, ok := req.(map[string]interface{}); ok {
		if rawRegion, ok := payload["region"].(string); ok && rawRegion != "" {
			region = rawRegion
		}
		query = strings.TrimSpace(stringValue(payload["query"]))
		if _, hasSingle := payload["single_main_genres"]; hasSingle {
			var built model.MysekaiTalkListRequest
			buf, err := json.Marshal(payload)
			if err != nil {
				return model.MysekaiTalkListRequest{}, err
			}
			if err := json.Unmarshal(buf, &built); err != nil {
				return model.MysekaiTalkListRequest{}, err
			}
			return built, nil
		}
	}
	if query == "" {
		return model.MysekaiTalkListRequest{}, fmt.Errorf("mysekai talk list requires character query")
	}

	merged := mergedMapFromUserData(c.userData)
	obtainedFixtureIDs := c.obtainedMysekaiFixtureIDs(merged, c.loadMasterDataMapByID("mysekaiBlueprints.json"))
	fixturesData := c.loadMasterDataList("mysekaiFixtures.json")
	fixtureMap := c.loadMasterDataMapByID("mysekaiFixtures.json")
	mainGenreMap := c.loadMasterDataMapByID("mysekaiFixtureMainGenres.json")
	gameCharUnitGroups := c.loadMasterDataMapByID("mysekaiGameCharacterUnitGroups.json")
	archiveGroups := c.loadMasterDataMapByID("characterArchiveMysekaiCharacterTalkGroups.json")
	conds := c.loadMasterDataList("mysekaiCharacterTalkConditions.json")
	condGroups := c.loadMasterDataList("mysekaiCharacterTalkConditionGroups.json")
	talks := c.loadMasterDataList("mysekaiCharacterTalks.json")

	cid, cuid := c.resolveTalkCharacter(query)
	if cuid == 0 {
		return model.MysekaiTalkListRequest{}, fmt.Errorf("mysekai talk list invalid character query: %s", query)
	}
	userTalkReads := map[int]bool{}
	for _, raw := range nestedList(merged, "userMysekaiCharacterTalks") {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		userTalkReads[intNumber(item["mysekaiCharacterTalkId"], 0)] = boolValue(item["isRead"])
	}

	type talkRead struct {
		fids     []int
		read     int
		total    int
		cuidsSet [][]int
		hasRead  bool
		cuids    []int
	}
	singleReads := map[string]*talkRead{}
	multiReadsMap := map[string]*talkRead{}
	aidReads := map[int]*talkRead{}

	condIDsByFixture := map[int][]int{}
	for _, cond := range conds {
		if stringValue(cond["mysekaiCharacterTalkConditionType"]) != "mysekai_fixture_id" {
			continue
		}
		fid := intNumber(cond["mysekaiCharacterTalkConditionTypeValue"], 0)
		if fid == 0 {
			continue
		}
		condIDsByFixture[fid] = append(condIDsByFixture[fid], intNumber(cond["id"], 0))
	}
	groupIDsByCond := map[int][]int{}
	for _, group := range condGroups {
		cidv := intNumber(group["mysekaiCharacterTalkConditionId"], 0)
		groupIDsByCond[cidv] = append(groupIDsByCond[cidv], intNumber(group["id"], 0))
	}
	talksByGroup := map[int][]map[string]interface{}{}
	for _, talk := range talks {
		gid := intNumber(talk["mysekaiCharacterTalkConditionGroupId"], 0)
		talksByGroup[gid] = append(talksByGroup[gid], talk)
	}

	for _, fixture := range fixturesData {
		fid := intNumber(fixture["id"], 0)
		if fid == 0 || stringValue(fixture["mysekaiFixtureType"]) == "gate" {
			continue
		}
		groupIDs := map[int]struct{}{}
		for _, condID := range condIDsByFixture[fid] {
			for _, gid := range groupIDsByCond[condID] {
				groupIDs[gid] = struct{}{}
			}
		}
		for gid := range groupIDs {
			for _, talk := range talksByGroup[gid] {
				tid := intNumber(talk["id"], 0)
				group := gameCharUnitGroups[intNumber(talk["mysekaiGameCharacterUnitGroupId"], 0)]
				if len(group) == 0 {
					continue
				}
				groupCuids := extractGroupCuids(group)
				if !containsInt(groupCuids, cuid) {
					continue
				}
				aid := intNumber(talk["characterArchiveMysekaiCharacterTalkGroupId"], 0)
				archive := archiveGroups[aid]
				if archive != nil && stringValue(archive["archiveDisplayType"]) != "normal" {
					continue
				}
				if _, ok := aidReads[aid]; !ok {
					aidReads[aid] = &talkRead{fids: []int{}, cuidsSet: [][]int{}}
				}
				if !containsInt(aidReads[aid].fids, fid) {
					aidReads[aid].fids = append(aidReads[aid].fids, fid)
				}
				aidReads[aid].cuids = groupCuids
				if userTalkReads[tid] {
					aidReads[aid].hasRead = true
				}
			}
		}
	}

	for _, item := range aidReads {
		sort.Ints(item.fids)
		keyParts := make([]string, 0, len(item.fids))
		for _, fid := range item.fids {
			keyParts = append(keyParts, strconv.Itoa(fid))
		}
		key := strings.Join(keyParts, " ")
		target := singleReads
		if len(item.cuids) > 1 {
			target = multiReadsMap
		}
		if _, ok := target[key]; !ok {
			target[key] = &talkRead{}
		}
		target[key].fids = item.fids
		target[key].total++
		if item.hasRead {
			target[key].read++
		} else if len(item.cuids) > 1 {
			dup := false
			for _, existing := range target[key].cuidsSet {
				if intsEqual(existing, item.cuids) {
					dup = true
					break
				}
			}
			if !dup {
				target[key].cuidsSet = append(target[key].cuidsSet, item.cuids)
			}
		}
	}

	groupedSingle := map[int][]model.MysekaiTalkFixtures{}
	for key, item := range singleReads {
		if item.total == item.read {
			continue
		}
		fids := parseIntTokens(key)
		if len(fids) == 0 {
			continue
		}
		fid := fids[0]
		fixture := fixtureMap[fid]
		mainGenreID := intNumber(fixture["mysekaiFixtureMainGenreId"], 0)
		var talkFixtures []model.MysekaiFixture
		for _, tfid := range fids {
			tfixture := fixtureMap[tfid]
			talkFixtures = append(talkFixtures, model.MysekaiFixture{
				ID:        tfid,
				ImagePath: fixtureThumbnailPath(tfixture),
				Obtained:  hasFixture(obtainedFixtureIDs, tfid),
			})
		}
		groupedSingle[mainGenreID] = append(groupedSingle[mainGenreID], model.MysekaiTalkFixtures{
			Fixtures:  talkFixtures,
			NoreadNum: item.total - item.read,
		})
	}
	var singleGenres []model.MysekaiSingleTalkMainGenre
	mainIDs := make([]int, 0, len(groupedSingle))
	for id := range groupedSingle {
		mainIDs = append(mainIDs, id)
	}
	sort.Ints(mainIDs)
	for _, id := range mainIDs {
		info := mainGenreMap[id]
		singleGenres = append(singleGenres, model.MysekaiSingleTalkMainGenre{
			Name:      stringValue(info["name"]),
			ImagePath: fmt.Sprintf("mysekai/icon/category_icon/%s.png", stringValue(info["assetbundleName"])),
			SubGenres: [][]model.MysekaiTalkFixtures{groupedSingle[id]},
		})
	}

	var multiReads []model.MysekaiTalkFixtures
	totalTalks := 0
	totalReads := 0
	for _, item := range singleReads {
		totalTalks += item.total
		totalReads += item.read
	}
	for key, item := range multiReadsMap {
		totalTalks += item.total
		totalReads += item.read
		if item.total == item.read {
			continue
		}
		fids := parseIntTokens(key)
		if len(fids) == 0 {
			continue
		}
		var iconGroups [][]string
		for _, cuids := range item.cuidsSet {
			var icons []string
			for _, gcuid := range cuids {
				icons = append(icons, fmt.Sprintf("chara_icon/%s.png", charaIconName(gcuid)))
			}
			iconGroups = append(iconGroups, icons)
		}
		var fixtures []model.MysekaiFixture
		for _, fid := range fids {
			fixture := fixtureMap[fid]
			fixtures = append(fixtures, model.MysekaiFixture{
				ID:        fid,
				ImagePath: fixtureThumbnailPath(fixture),
				Obtained:  hasFixture(obtainedFixtureIDs, fid),
			})
		}
		multiReads = append(multiReads, model.MysekaiTalkFixtures{
			Fixtures:             fixtures,
			NoreadNum:           item.total - item.read,
			CharacterIDs:        item.cuidsSet,
			CharaIconPathGroups: iconGroups,
		})
	}
	sort.SliceStable(multiReads, func(i, j int) bool {
		if len(multiReads[i].Fixtures) != len(multiReads[j].Fixtures) {
			return len(multiReads[i].Fixtures) > len(multiReads[j].Fixtures)
		}
		if len(multiReads[i].Fixtures) == 0 || len(multiReads[j].Fixtures) == 0 {
			return false
		}
		return multiReads[i].Fixtures[0].ID < multiReads[j].Fixtures[0].ID
	})

	progress := fmt.Sprintf("未读对话家具列表 - 进度: %d/%d (%.1f%%)", totalReads, totalTalks, percent(totalReads, totalTalks))
	prompt := "*仅展示未读对话家具，灰色表示未获得蓝图"
	reqOut := model.MysekaiTalkListRequest{
		Profile:          c.mysekaiProfileCard(region, merged),
		SDImagePath:      fmt.Sprintf("character/character_sd_l/chr_sp_%d.png", cuid),
		ProgressMessage:  &progress,
		PromptMessage:    &prompt,
		ShowID:           true,
		SingleMainGenres: singleGenres,
		MultiReads:       multiReads,
	}
	_ = cid
	return reqOut, nil
}

func (c *MysekaiController) resolveTalkCharacter(query string) (int, int) {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" || c.masterData == nil {
		return 0, 0
	}
	nicknames := c.masterData.GetNicknames()
	fields := strings.Fields(query)
	for _, field := range fields {
		if cid, ok := nicknames[field]; ok {
			for _, item := range c.loadMasterDataList("gameCharacterUnits.json") {
				if intNumber(item["gameCharacterId"], 0) == cid {
					return cid, intNumber(item["id"], 0)
				}
			}
		}
	}
	return 0, 0
}

func extractGroupCuids(group map[string]interface{}) []int {
	var cuids []int
	for i := 1; i <= 9; i++ {
		key := fmt.Sprintf("gameCharacterUnitId%d", i)
		if id := intNumber(group[key], 0); id != 0 {
			cuids = append(cuids, id)
		}
	}
	return cuids
}

func containsInt(items []int, target int) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func hasFixture(obtained map[int]struct{}, fid int) bool {
	if len(obtained) == 0 {
		return true
	}
	_, ok := obtained[fid]
	return ok
}

func percent(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) * 100 / float64(b)
}

func intsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isMusicAvailableNow(windows []map[string]interface{}, nowMs int64) bool {
	for _, item := range windows {
		start := int64Number(item["startAt"], 0)
		end := int64Number(item["endAt"], 0)
		if start <= nowMs && nowMs <= end {
			return true
		}
	}
	return false
}
