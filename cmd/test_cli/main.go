package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"Haruki-Service-API/internal/apiutils"
	"Haruki-Service-API/internal/config"
	"Haruki-Service-API/internal/controller"
	"Haruki-Service-API/internal/model"
	"Haruki-Service-API/internal/service"
	"Haruki-Service-API/pkg/asset"

	_ "github.com/lib/pq"
)

type cliEnv struct {
	masterdata          *service.MasterDataService
	cardController      *controller.CardController
	musicController     *controller.MusicController
	gachaController     *controller.GachaController
	eventController     *controller.EventController
	educationController *controller.EducationController
	honorController     *controller.HonorController
	profileController   *controller.ProfileController
	stampController     *controller.StampController
	miscController      *controller.MiscController
	scoreController     *controller.ScoreController
	deckController      *controller.DeckController
	skController        *controller.SkController
	mysekaiController   *controller.MysekaiController
	cardParser          *service.CardParser
	eventParser         *service.EventParser
	eventSearch         *service.EventSearchService
	musicSearch         *service.MusicSearchService
	userData            *service.UserDataService
	resolver            *service.GlobalCommandResolver
}

type scenario struct {
	Name        string `json:"name"`
	Mode        string `json:"mode"`
	Cmd         string `json:"cmd"`
	Description string `json:"description"`
}

var globalOutputDir string

func main() {
	modePtr := flag.String("mode", "auto", "Mode: auto/detail/card-detail, card-list, card-box, music (detail), music-brief, music-list, music-progress, music-chart, music-reward-detail, music-reward-basic, gacha-list, gacha-detail, event-detail, event-list, event-record, education-* (challenge/power/area/bonds/leader), honor, profile, stamp-list, misc-chara-birthday, score-control/score-custom-room/score-music-meta/score-music-board, deck-recommend/deck-recommend-auto, sk-*, mysekai-*")
	cmdPtr := flag.String("cmd", "", "Command payload, e.g. '/闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閻愵剙鍔ょ紓宥咃躬瀵鎮㈤崗灏栨嫽闁诲酣娼ф竟濠偽ｉ鍓х＜闁绘劦鍓欓崝銈囩磽瀹ュ拑韬€殿喖顭烽幃銏ゅ礂鐏忔牗瀚介梺璇查叄濞佳勭珶婵犲伣锝夘敊閸撗咃紲闂佺粯鍔﹂崜娆撳礉閵堝洨纾界€广儱鎷戦煬顒傗偓娈垮枛椤兘骞冮姀銈呯閻忓繑鐗楃€氫粙姊虹拠鏌ュ弰婵炰匠鍕彾濠电姴浼ｉ敐澶樻晩闁告挆鍜冪床闂備胶绮崝锕傚礈濞嗘挸绀夐柕鍫濇川绾剧晫鈧箍鍎遍幏鎴︾叕椤掑倵鍋撳▓鍨灈妞ゎ厾鍏橀獮鍐閵堝懐顦ч柣蹇撶箲閻楁鈧矮绮欏铏规嫚閺屻儱寮板┑鐐板尃閸曨厾褰炬繝鐢靛Т娴硷綁鏁愭径妯绘櫓闂佸憡鎸嗛崪鍐簥闂傚倷娴囬鏍垂鎼淬劌绀冮柨婵嗘閻﹂亶姊婚崒娆掑厡妞ゃ垹锕ら埢宥夊即閵忕姷顔夐梺鎼炲労閸撴瑩鎮橀幎鑺ョ厸闁告劑鍔庢晶鏇犵磼閳ь剟宕橀埞澶哥盎闂婎偄娲ゅù鐑剿囬敃鈧湁婵犲﹤鐗忛悾娲煛鐏炶濡奸柍瑙勫灴瀹曞崬鈻庤箛鎾寸槗缂傚倸鍊烽梽宥夊礉瀹€鍕ч柟闂寸閽冪喖鏌ｉ弬鍨倯闁稿骸鐭傞弻娑樷攽閸曨偄濮㈤悶姘剧畵濮婄粯鎷呴崨濠冨創闂佹椿鍘奸ˇ杈╂閻愬鐟归柍褜鍓熸俊瀛樻媴閸撳弶寤洪梺閫炲苯澧存鐐插暙閳诲酣骞樺畷鍥跺晣婵＄偑鍊栭幐楣冨闯閵夈儙娑滎樄婵﹤顭峰畷鎺戔枎閹寸姷宕叉繝鐢靛仒閸栫娀宕楅悙顒傗槈闁宠閰ｉ獮瀣倷鐎涙﹩鍞堕梻鍌欑濠€閬嶅磿閵堝鈧啴骞囬鍓ь槸闂佸搫绉查崝搴ｅ姬閳ь剟姊婚崒姘卞濞撴碍顨婂畷鏇㈠箛閻楀牏鍘搁梺鍛婁緱閸犳岸宕ｉ埀顒勬⒑閸濆嫭婀扮紒瀣灴閸┿儲寰勯幇顒傤攨闂佺粯鍔曞Ο濠傤焽缂佹ü绻嗛柣鎰典簻閳ь剚鍨垮畷鏇㈠蓟閵夛箑娈炴俊銈忕到閸燁偊鎮″鈧弻鐔衡偓鐢殿焾娴犳粎绱掗悩闈涒枅婵﹨娅ｇ划娆撳礌閳╁啯鏆版俊鐐€戦崝灞轿涘┑瀣祦闁割偁鍎辨儫闂佸啿鎼崐鎼佸焵椤掆偓椤兘寮婚敃鈧灒濞撴凹鍨辨闂備焦瀵х粙鎺楁儎椤栨凹娼栭柧蹇撴贡绾惧吋淇婇姘儓妞ゎ偄閰ｅ铏圭矙鐠恒劍妲€闂佺锕ョ换鍌炴偩閻戣棄绠ｉ柣姗嗗亜娴滈箖鏌ㄥ┑鍡涱€楅柡瀣枛閺岋綁骞樼捄鐑樼€炬繛锝呮搐閿曨亪銆佸☉妯锋斀闁糕剝顨嗛崕顏呯節閻㈤潧袥闁稿鎸搁湁闁绘ê妯婇崕鎰版煟閹惧啿鏆熼柟鑼焾椤劑宕煎┑鍫Н婵犵數鍋為崹鍓佸枈瀹ュ鏁冮柤鎭掑劜閸欏繑鎱ㄥΔ鈧Λ妤佹櫠婵犳碍鐓曢柕鍫濇缁€瀣煛瀹€鈧崰鎾舵閹烘顫呴柣妯虹－娴滎亝淇婇悙顏勨偓銈夊磻閸曨垰绠犳慨妞诲亾鐎殿喖顭烽弫鎰板川閸屾粌鏋涢柟绛圭節婵″爼宕ㄩ娆愬亝闂傚倸鍊烽悞锕傚箖閸洖纾挎い鏍仜缁€澶愬箹濞ｎ剙濡奸柛灞诲姂濮婃椽顢楅埀顒傜矓閻㈢鐓曢柟杈鹃檮閻撴盯鏌涚仦涔咁亪宕濆鍕闁告侗鍘介崵鍥煛鐏炲墽娲撮柟顔哄灲瀵爼骞嬪┑鎰偓瀵哥磽閸屾瑧顦﹂柛濠傛贡閺侇噣鏁撻悩鍙夌€梺绋跨灱閸嬫稓绮堥崼銏″枑闊洦绋掗弲顒佺節婵犲倸鏋ゆ繛鎾愁煼閺屾洟宕煎┑鍥舵！闂佹娊鏀辩敮锟犲蓟閻斿吋鎯炴い鎰剁到闂夊秹姊洪崫鍕櫣闁绘牕銈搁悰顕€骞樼拠鑼唹濠电偞鍨堕敃銏ゅ矗閸曨厸鍋撶憴鍕闁稿骸銈哥瘬闁稿瞼鍋為ˉ濠冦亜閹烘埈妲稿褎鎸抽弻鈥崇暆閳ь剟宕伴弽顓溾偓浣糕枎閹炬潙浠奸柣蹇曞仦閸庡啿鈻嶅顓濈箚闁绘劦浜滈埀顒佸灴瀹曞綊宕崟搴㈢洴瀹曟﹢濡歌濞堥箖姊虹紒妯烩拻闁告鍕姅闂傚倷绶氬褔藝椤撱垹纾归柡鍥ｆ嚍婢跺娼╅柤鍝ヮ暯閹风粯绻涙潏鍓у埌闁硅绻濋獮鍡涘醇閵忋垻锛滈柣搴岛閺呮繄绮ｉ弮鍌楀亾鐟欏嫭纾搁柛銊ょ矙楠炲﹪寮介鐐靛幐闂佸憡鍔︽禍鐐哄礉鐠鸿　鏀介柣姗嗗枛閻忚鲸绻涙径瀣创鐎殿喚鏁婚、妤呭礋椤愩倗宕堕梻浣告贡婢ф绔熼崱娑樼煑闁瑰墽绮悡鏇㈡煏婢舵稓鍒板┑锛勬櫕缁辨帡鍩€椤掑嫬鐒垫い鎺戝閳锋垿鏌熼懖鈺佷粶濠碉紕鏅槐鎺旀嫚閹绘巻鍋撻崸妤€鏄ラ柣鎰惈缁狅綁鏌ㄩ弮鍥棄濞存粌缍婇弻锝嗘償閳╁啳纭€婵犳鍠楅幐瀹犳＂濠电娀娼ч悧蹇曞閽樺褰掓晲閸涱喗鍎撳銈呭閹稿墽妲愰幘鎰佸悑闁告洦鍘鹃悰銏ゆ倵濞堝灝鏋涙い顓犲厴瀵偊骞囬弶鍨獩闂佺鏈划宀勫几濞戙垺鐓欐い鏃傛櫕閻﹪鎽堕敐澶嬧拺闁割煈鍣崕鎴澝瑰鍛粶闁宠鍨块幃鈺佲枔閹稿孩鐦滄繝纰樺墲瑜板啴骞婂Ο铏规殾闁硅揪绠戝婵囥亜閺嶃劏澹橀柣蹇擄躬濮婃椽宕烽鐐板闂佸憡鎸婚懝楣冣€﹂崶顒€鍐€妞ゆ挾鍠撻崣鍕椤愩垺澶勯柟姝屽吹閹峰綊鏁撻悩宕囧幗濡炪倖鎸鹃崰鎰暦瀹€鍕厸閻忕偛澧藉ú鎾煕閳轰礁顏€规洘锕㈤幃娆擃敆閸屾稒顔旀繝纰夌磿閸嬫垿宕愰弴鐘冲床闁规壆澧楅崑銈夋煏婵炑冨鎼村﹤鈹戦悙鏉戠仧闁搞劌缍婇弻瀣炊椤掍胶鍘介梺鍝勫€圭€笛囧箟閸涘浜滈柨鏂跨仢瀹撳棙鎱ㄦ繝鍐┿仢鐎规洦鍋婂畷鐔碱敇閻樻彃蝎缂傚倸鍊搁崐鍝ョ矓瀹曞洦顐芥慨妯垮煐閸嬫ɑ銇勯弬鎸庮潔闁绘棁妗ㄩ悞濠囨煛瀹擃喖鍊搁ˉ姘節閻㈤潧浠﹂柛銊ョ埣閹兘顢涢悙鏉戔偓鍧楁煠閹帒鍔樺ù婊勭矒閺屻劑寮幐搴㈠創婵犵绱曢弲顐ゆ閹烘梻纾兼俊顖氬悑閸掓盯姊烘潪鎵妽闁告梹鐟ラ悾鐑藉Ω閳哄﹥鏅╅梺鍏间航閸庨亶鍩€椤掑倸鍘存慨濠勭帛閹峰懘鎮烽柇锕€娈濇繝鐢靛仩椤曟粓姊介崟顓犵焿鐎广儱鎳夐弨浠嬫煕閵夈垺鏉归柡鈧搹顐ょ瘈闁汇垽娼у瓭濠电偛鐪伴崐婵嬪蓟鐎ｎ喖鐐婃い鎺嶈兌閸樻悂姊洪幖鐐插姉闁哄懏绋掔粋鎺戭煥閸涱垳锛滈梺褰掑亰閸橀箖鍩ユ径瀣ㄤ簻妞ゅ繐瀚弳锝呪攽閳ュ磭鍩ｇ€规洖宕灒闁绘垶蓱椤斿倿姊婚崒娆戭槮闁硅姤绮嶉幈銊╂偨缁嬭法顦┑鐐叉閹稿爼鍩€椤戣法顦﹂柍璇查叄楠炴ê鐣烽崶鑸敌у┑锛勫亼閸婃牠宕濆Δ鍛獥闁圭増婢樺Λ姗€鏌嶈閸撶喎顫忕紒妯诲闁惧繒鎳撶粭锟犳⒑閸涘﹥鈷愭繛鍙夌墳閻忔帗绻濋悽闈浶㈡繛灞傚€濆畷銉ㄣ亹閹烘挾鍘介梺鍝勫€藉▔鏇炩枔闁秵鐓涢柍褜鍓熼幊鐐哄Ψ閿濆嫮鐩庨梻浣告惈閸燁偄煤閵堝棛顩茬憸鐗堝笒閺嬩線鏌涢鐘插姕闁稿﹦鏁婚幃宄扳枎韫囨搩浠剧紓浣插亾闁逞屽墴濮婅櫣绱掑Ο鐓庘吂闂佸憡鑹鹃鍡欑矚鏉堛劎绡€闁稿被鍊栧銊╂倵閸忓浜炬繝鐢靛Т閸烆參鍩￠崒娆戠畾?190'")
	scenarioPtr := flag.String("scenario", "", "Run multiple commands; use 'all' for built-in regression or provide a JSON file path")
	flag.Parse()

	configPath := "../../configs/configs.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = "configs/configs.yaml"
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			configPath = "configs.yaml"
		}
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	globalOutputDir = cfg.DrawingAPI.OutputDir

	cloudClients, err := apiutils.InitCloudClients(cfg.HarukiCloud, slog.Default())
	if err != nil {
		slog.Error("Failed to initialize Haruki Cloud clients", "error", err)
		os.Exit(1)
	}
	defer cloudClients.Close()
	cloudService := service.NewCloudService(cloudClients.Sekai)

	slog.Info("Initializing services...")
	masterdata := service.NewMasterDataService(cfg.MasterData.Dir, "JP")
	if err := masterdata.LoadAll(); err != nil {
		slog.Error("Failed to load masterdata", "error", err)
		os.Exit(1)
	}

	assetHelper := asset.NewAssetHelper(cfg.Assets.Dir, cfg.Assets.LegacyDirs)

	var userData *service.UserDataService
	if cfg.UserData.Path != "" {
		data, err := service.NewUserDataService(cfg.UserData.Path, assetHelper.Primary(), masterdata, masterdata.GetRegion())
		if err != nil {
			slog.Warn("Failed to load user data", "path", cfg.UserData.Path, "error", err)
		} else {
			userData = data
		}
	}

	drawing := service.NewDrawingService(cfg.DrawingAPI.BaseURL, cfg.DrawingAPI.Timeout, cfg.DrawingAPI.RetryCount, assetHelper.Roots())
	deckRecommender := service.NewDeckRecommenderService(cfg.DeckRecommend)

	nicknames := masterdata.GetNicknames()
	cardParser := service.NewCardParser(nicknames)
	cloudRegion := strings.TrimSpace(cfg.HarukiCloud.Region)
	if cloudRegion == "" {
		cloudRegion = masterdata.GetRegion()
	}
	secondaryRegion := strings.TrimSpace(cfg.HarukiCloud.SecondaryRegion)
	if secondaryRegion == "" {
		secondaryRegion = "jp"
	}

	masterCardSource := service.NewMasterDataCardSource(masterdata)
	var cardSource service.CardDataSource
	if cloudClients.Sekai != nil {
		cardSource = service.NewCloudCardSource(cloudClients.Sekai, cloudRegion)
	}
	if cardSource == nil {
		cardSource = masterCardSource
	}
	var secondaryCardSource service.CardDataSource
	if cloudClients.Sekai != nil && secondaryRegion != "" {
		secondaryCardSource = service.NewCloudCardSource(cloudClients.Sekai, secondaryRegion)
	}
	if secondaryCardSource == nil {
		secondaryCardSource = masterCardSource
	}

	var eventSource service.EventDataSource
	if cloudClients.Sekai != nil {
		eventSource = service.NewCloudEventSource(cloudClients.Sekai, cloudRegion)
	}
	if eventSource == nil {
		eventSource = service.NewMasterDataEventSource(masterdata)
	}
	var secondaryEventSource service.EventDataSource
	if cloudClients.Sekai != nil && secondaryRegion != "" && !strings.EqualFold(secondaryRegion, cloudRegion) {
		secondaryEventSource = service.NewCloudEventSource(cloudClients.Sekai, secondaryRegion)
	}

	cardSearchService := service.NewCardSearchService(cardSource, cardParser)
	eventParser := service.NewEventParser(nicknames)
	eventSearch := service.NewEventSearchService(eventSource, eventParser)
	musicParser := service.NewMusicParser(masterdata)
	musicSearch := service.NewMusicSearchService(masterdata, musicParser)

	musicSource := service.MusicDataSource(service.NewMasterDataMusicSource(masterdata))
	if cloudClients.Sekai != nil {
		if src := service.NewCloudMusicSource(cloudClients.Sekai, cloudRegion); src != nil {
			musicSource = src
		}
	}
	masterGachaSource := service.NewMasterDataGachaSource(masterdata)
	var gachaSource service.GachaDataSource
	if cloudClients.Sekai != nil {
		gachaSource = service.NewCloudGachaSource(cloudClients.Sekai, cloudRegion)
	}
	if gachaSource == nil {
		gachaSource = masterGachaSource
	}
	masterHonorSource := service.NewMasterDataHonorSource(masterdata)
	var honorSource service.HonorDataSource
	if cloudClients.Sekai != nil {
		honorSource = service.NewCloudHonorSource(cloudClients.Sekai, cloudRegion)
	}
	if honorSource == nil {
		honorSource = masterHonorSource
	}
	masterProfileSource := service.NewMasterDataProfileSource(masterdata)
	var profileSource service.ProfileDataSource
	if cloudClients.Sekai != nil {
		profileSource = service.NewCloudProfileSource(cloudClients.Sekai, cloudRegion)
	}
	if profileSource == nil {
		profileSource = masterProfileSource
	}
	masterEducationSource := service.NewMasterDataEducationSource(masterdata)
	var educationSource service.EducationDataSource
	if cloudClients.Sekai != nil {
		educationSource = service.NewCloudEducationSource(cloudClients.Sekai, cloudRegion)
	}
	if educationSource == nil {
		educationSource = masterEducationSource
	}

	cardController := controller.NewCardController(cardSource, secondaryCardSource, eventSource, masterdata, drawing, cardSearchService, cfg.DrawingAPI.BaseURL, assetHelper, userData)
	if secondaryEventSource != nil {
		cardController.RegisterEventSource(secondaryEventSource)
	}
	musicController := controller.NewMusicController(musicSource, drawing, cfg.DrawingAPI.BaseURL, assetHelper, userData)
	if cloudClients.Sekai != nil {
		for _, region := range []string{"jp", "en", "tw", "kr", "cn", secondaryRegion} {
			normalized := strings.ToLower(strings.TrimSpace(region))
			if normalized == "" || strings.EqualFold(normalized, cloudRegion) {
				continue
			}
			if src := service.NewCloudMusicSource(cloudClients.Sekai, normalized); src != nil {
				musicController.RegisterSource(src)
			}
		}
	}
	gachaController := controller.NewGachaController(gachaSource, drawing, cfg.DrawingAPI.BaseURL, assetHelper)
	if cloudClients.Sekai != nil && secondaryRegion != "" && !strings.EqualFold(secondaryRegion, cloudRegion) {
		if src := service.NewCloudGachaSource(cloudClients.Sekai, secondaryRegion); src != nil {
			gachaController.RegisterSource(src)
		}
	}
	honorController := controller.NewHonorController(honorSource, drawing, assetHelper)
	if cloudClients.Sekai != nil && secondaryRegion != "" && !strings.EqualFold(secondaryRegion, cloudRegion) {
		if src := service.NewCloudHonorSource(cloudClients.Sekai, secondaryRegion); src != nil {
			honorController.RegisterSource(src)
		}
	}
	eventController := controller.NewEventController(eventSource, drawing, cfg.DrawingAPI.BaseURL, assetHelper, cloudService)
	if secondaryEventSource != nil {
		eventController.RegisterSource(secondaryEventSource)
	}
	profileController := controller.NewProfileController(profileSource, drawing, assetHelper)
	if cloudClients.Sekai != nil && secondaryRegion != "" && !strings.EqualFold(secondaryRegion, cloudRegion) {
		if src := service.NewCloudProfileSource(cloudClients.Sekai, secondaryRegion); src != nil {
			profileController.RegisterSource(src)
		}
	}
	educationController := controller.NewEducationController(educationSource, drawing, assetHelper, userData)
	if cloudClients.Sekai != nil && secondaryRegion != "" && !strings.EqualFold(secondaryRegion, cloudRegion) {
		if src := service.NewCloudEducationSource(cloudClients.Sekai, secondaryRegion); src != nil {
			educationController.RegisterSource(src)
		}
	}
	masterStampSource := service.NewMasterDataStampSource(masterdata)
	var stampSource service.StampDataSource
	if cloudClients.Sekai != nil {
		stampSource = service.NewCloudStampSource(cloudClients.Sekai, cloudRegion)
	}
	if stampSource == nil {
		stampSource = masterStampSource
	}
	stampController := controller.NewStampController(stampSource, drawing, assetHelper)
	if cloudClients.Sekai != nil && secondaryRegion != "" && !strings.EqualFold(secondaryRegion, cloudRegion) {
		if src := service.NewCloudStampSource(cloudClients.Sekai, secondaryRegion); src != nil {
			stampController.RegisterSource(src)
		}
	}
	miscController := controller.NewMiscController(drawing)
	scoreController := controller.NewScoreController(drawing)
	deckController := controller.NewDeckController(drawing, cardSource, eventSource, assetHelper, userData, deckRecommender)
	skController := controller.NewSkController(drawing)
	mysekaiController := controller.NewMysekaiController(drawing)

	env := &cliEnv{
		masterdata:          masterdata,
		cardController:      cardController,
		musicController:     musicController,
		gachaController:     gachaController,
		honorController:     honorController,
		profileController:   profileController,
		stampController:     stampController,
		miscController:      miscController,
		scoreController:     scoreController,
		deckController:      deckController,
		skController:        skController,
		mysekaiController:   mysekaiController,
		cardParser:          cardParser,
		eventController:     eventController,
		educationController: educationController,
		eventParser:         eventParser,
		eventSearch:         eventSearch,
		musicSearch:         musicSearch,
		userData:            userData,
		resolver:            service.NewGlobalCommandResolver(nicknames),
	}

	if *scenarioPtr != "" {
		if err := env.runScenario(*scenarioPtr); err != nil {
			slog.Error("Scenario failed", "scenario", *scenarioPtr, "error", err)
			os.Exit(1)
		}
		return
	}

	if err := env.runMode(*modePtr, *cmdPtr); err != nil {
		slog.Error("Mode execution failed", "mode", *modePtr, "cmd", *cmdPtr, "error", err)
		os.Exit(1)
	}
}

func (env *cliEnv) runScenario(name string) error {
	scenarios, err := env.resolveScenario(name)
	if err != nil {
		return err
	}
	fmt.Printf("Running %d scenario(s)\n", len(scenarios))
	for idx, sc := range scenarios {
		fmt.Printf("\n[%d/%d] %s - %s\n", idx+1, len(scenarios), sc.Name, sc.Description)
		if err := env.runMode(sc.Mode, sc.Cmd); err != nil {
			return fmt.Errorf("scenario %s failed: %w", sc.Name, err)
		}
	}
	fmt.Println("\nAll scenarios finished successfully.")
	return nil
}

func (env *cliEnv) resolveScenario(name string) ([]scenario, error) {
	lower := strings.ToLower(strings.TrimSpace(name))
	switch {
	case lower == "all":
		return defaultScenarios(env)
	case strings.HasSuffix(lower, ".json"):
		return loadScenarioFile(name)
	default:
		return nil, fmt.Errorf("unknown scenario: %s", name)
	}
}

func defaultScenarios(env *cliEnv) ([]scenario, error) {
	slots := []struct {
		Name string
		Mode string
		Desc string
	}{
		{"card-detail", "card-detail", "Card detail"},
		{"card-list", "card-list", "Card list query"},
		{"card-box", "card-box", "Card box"},
		{"music-detail", "music", "Music detail"},
		{"music-brief", "music-brief", "Music brief"},
		{"music-list", "music-list", "Music list"},
		{"music-progress", "music-progress", "Music progress"},
		{"music-chart", "music-chart", "Music chart"},
		{"gacha-list", "gacha-list", "Gacha list"},
		{"gacha-detail", "gacha-detail", "Gacha detail"},
		{"event-detail", "event-detail", "Event detail"},
		{"event-list", "event-list", "Event list"},
	}
	var scenarios []scenario
	for _, slot := range slots {
		cmd, err := env.defaultCommand(slot.Mode)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, scenario{Name: slot.Name, Mode: slot.Mode, Cmd: cmd, Description: slot.Desc})
	}
	return scenarios, nil
}

func loadScenarioFile(path string) ([]scenario, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	var scenarios []scenario
	if err := json.Unmarshal(data, &scenarios); err != nil {
		return nil, err
	}
	if len(scenarios) == 0 {
		return nil, fmt.Errorf("scenario file %s is empty", path)
	}
	return scenarios, nil
}

func (env *cliEnv) runMode(mode string, cmd string) error {
	normalized := strings.ToLower(strings.TrimSpace(mode))

	if normalized == "auto" {
		res, err := env.resolver.Resolve(cmd)
		if err != nil {
			return err
		}
		return env.handleResolvedCommand(res)
	}

	switch normalized {
	case "detail", "card-detail":
		payload, err := env.ensureCommand("card-detail", cmd)
		if err != nil {
			return err
		}
		return testCardDetail(env.cardController, env.cardParser, payload)
	case "list", "card-list":
		if strings.TrimSpace(cmd) == "" {
			return testCardListHardcoded(env.cardController)
		}
		return testCardListDynamic(env.cardController, cmd)
	case "box", "card-box":
		payload, err := env.ensureCommand("card-box", cmd)
		if err != nil {
			return err
		}
		return testCardBox(env.cardController, payload)
	case "music", "music-detail":
		payload, err := env.ensureCommand("music", cmd)
		if err != nil {
			return err
		}
		return testMusicDetail(env.musicController, env.musicSearch, payload)
	case "music-brief":
		payload, err := env.ensureCommand("music-brief", cmd)
		if err != nil {
			return err
		}
		return testMusicBriefList(env.musicController, payload)
	case "music-list":
		payload, err := env.ensureCommand("music-list", cmd)
		if err != nil {
			return err
		}
		return testMusicList(env.musicController, payload)
	case "music-progress":
		payload, err := env.ensureCommand("music-progress", cmd)
		if err != nil {
			return err
		}
		return testMusicProgress(env.musicController, payload)
	case "music-chart":
		payload, err := env.ensureCommand("music-chart", cmd)
		if err != nil {
			return err
		}
		return testMusicChart(env.musicController, payload)
	case "music-reward-detail":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("music-reward-detail requires -cmd JSON file path")
		}
		return testMusicRewardsDetail(env.musicController, cmd)
	case "music-reward-basic":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("music-reward-basic requires -cmd JSON file path")
		}
		return testMusicRewardsBasic(env.musicController, cmd)
	case "gacha-list":
		payload, err := env.ensureCommand("gacha-list", cmd)
		if err != nil {
			return err
		}
		return testGachaList(env.gachaController, payload)
	case "gacha-detail":
		payload, err := env.ensureCommand("gacha-detail", cmd)
		if err != nil {
			return err
		}
		return testGachaDetail(env, payload)
	case "event-detail":
		payload, err := env.ensureCommand("event-detail", cmd)
		if err != nil {
			return err
		}
		return testEventDetail(env.eventController, env.eventSearch, payload)
	case "event-list":
		payload, err := env.ensureCommand("event-list", cmd)
		if err != nil {
			return err
		}
		return testEventList(env.eventController, env.eventParser, payload)
	case "event-record":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("event-record mode requires -cmd JSON file path")
		}
		return testEventRecord(env.eventController, cmd)
	case "education-challenge":
		return testEducationChallengeLive(env.educationController, cmd)
	case "education-power":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("education-power mode requires -cmd JSON file path")
		}
		return testEducationPowerBonus(env.educationController, cmd)
	case "education-area":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("education-area mode requires -cmd JSON file path")
		}
		return testEducationAreaItem(env.educationController, cmd)
	case "education-bonds":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("education-bonds mode requires -cmd JSON file path")
		}
		return testEducationBonds(env.educationController, cmd)
	case "education-leader":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("education-leader mode requires -cmd JSON file path")
		}
		return testEducationLeaderCount(env.educationController, cmd)
	case "honor":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("honor mode requires -cmd JSON file path")
		}
		return testHonorGenerate(env.honorController, cmd)
	case "profile":
		return testProfileGenerate(env.profileController, env.userData, cmd)
	case "stamp-list":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("stamp-list mode requires -cmd JSON file path")
		}
		return testStampList(env.stampController, cmd)
	case "misc-chara-birthday":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("misc-chara-birthday mode requires -cmd JSON file path")
		}
		return testMiscCharaBirthday(env.miscController, cmd)
	case "score-control":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("score-control mode requires -cmd JSON file path")
		}
		return testScoreControl(env.scoreController, cmd)
	case "score-custom-room":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("score-custom-room mode requires -cmd JSON file path")
		}
		return testScoreCustomRoom(env.scoreController, cmd)
	case "score-music-meta":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("score-music-meta mode requires -cmd JSON file path")
		}
		return testScoreMusicMeta(env.scoreController, cmd)
	case "score-music-board":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("score-music-board mode requires -cmd JSON file path")
		}
		return testScoreMusicBoard(env.scoreController, cmd)
	case "deck-recommend":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("deck-recommend mode requires -cmd JSON file path")
		}
		return testDeckRecommend(env.deckController, cmd)
	case "deck-recommend-auto":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("deck-recommend-auto mode requires -cmd JSON file path")
		}
		return testDeckRecommendAuto(env.deckController, cmd)
	case "sk-line":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("sk-line mode requires -cmd JSON file path")
		}
		return testSKLine(env.skController, cmd)
	case "sk-query":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("sk-query mode requires -cmd JSON file path")
		}
		return testSKQuery(env.skController, cmd)
	case "sk-check-room":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("sk-check-room mode requires -cmd JSON file path")
		}
		return testSKCheckRoom(env.skController, cmd)
	case "sk-speed":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("sk-speed mode requires -cmd JSON file path")
		}
		return testSKSpeed(env.skController, cmd)
	case "sk-player-trace":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("sk-player-trace mode requires -cmd JSON file path")
		}
		return testSKPlayerTrace(env.skController, cmd)
	case "sk-rank-trace":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("sk-rank-trace mode requires -cmd JSON file path")
		}
		return testSKRankTrace(env.skController, cmd)
	case "sk-winrate":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("sk-winrate mode requires -cmd JSON file path")
		}
		return testSKWinrate(env.skController, cmd)
	case "mysekai-resource":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("mysekai-resource mode requires -cmd JSON file path")
		}
		return testMysekaiResource(env.mysekaiController, cmd)
	case "mysekai-fixture-list":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("mysekai-fixture-list mode requires -cmd JSON file path")
		}
		return testMysekaiFixtureList(env.mysekaiController, cmd)
	case "mysekai-fixture-detail":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("mysekai-fixture-detail mode requires -cmd JSON file path")
		}
		return testMysekaiFixtureDetail(env.mysekaiController, cmd)
	case "mysekai-door-upgrade":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("mysekai-door-upgrade mode requires -cmd JSON file path")
		}
		return testMysekaiDoorUpgrade(env.mysekaiController, cmd)
	case "mysekai-music-record":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("mysekai-music-record mode requires -cmd JSON file path")
		}
		return testMysekaiMusicRecord(env.mysekaiController, cmd)
	case "mysekai-talk-list":
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("mysekai-talk-list mode requires -cmd JSON file path")
		}
		return testMysekaiTalkList(env.mysekaiController, cmd)
	default:
		return fmt.Errorf("unknown mode: %s", mode)
	}
}

func (env *cliEnv) handleResolvedCommand(res *service.ResolvedCommand) error {
	if res.IsHelp {
		fmt.Println("Haruki Command Help:")
		fmt.Println("  /card <mnr> [-r jp/en/cn] - card detail")
		fmt.Println("  /music <id/name> [-r jp/en/cn] - music detail")
		fmt.Println("  /event [current/id/name] - event detail")
		fmt.Println("  /sk [uid/rank/@user] - event record detail")
		return nil
	}

	if res.Region != "" {
		slog.Info("Switching region", "target", res.Region)
	}

	var err error
	switch res.Module {
	case service.ModuleCard:
		switch res.Mode {
		case "gacha-list":
			err = testGachaList(env.gachaController, res.Query)
		case "card-box":
			err = testCardBox(env.cardController, res.Query)
		case "card-list":
			err = testCardListDynamic(env.cardController, res.Query)
		default:
			err = testCardDetail(env.cardController, env.cardParser, res.Query)
		}
	case service.ModuleMusic:
		switch res.Mode {
		case "music-chart":
			err = testMusicChart(env.musicController, res.Query)
		case "music-list":
			err = testMusicList(env.musicController, res.Query)
		case "music-progress":
			err = testMusicProgress(env.musicController, res.Query)
		default:
			err = testMusicDetail(env.musicController, env.musicSearch, res.Query)
		}
	case service.ModuleEvent:
		switch res.Mode {
		case "event-list":
			err = testEventList(env.eventController, env.eventParser, res.Query)
		default:
			err = testEventDetail(env.eventController, env.eventSearch, res.Query)
		}
	case service.ModuleProfile:
		err = testProfileGenerate(env.profileController, env.userData, res.Query)
	case service.ModuleGacha:
		switch res.Mode {
		case "gacha":
			err = testGachaDetail(env, res.Query)
			if err != nil {
				err = testGachaList(env.gachaController, res.Query)
			}
		default:
			err = testGachaList(env.gachaController, res.Query)
		}
	case service.ModuleDeck:
		switch res.Mode {
		case "deck-event":
			err = testDeckRecommendAuto(env.deckController, res.Query)
		case "deck-no-event":
			err = testDeckRecommendAuto(env.deckController, res.Query)
		case "deck-bonus":
			err = testDeckRecommendAuto(env.deckController, res.Query)
		case "deck-challenge":
			err = testDeckRecommendAuto(env.deckController, res.Query)
		case "deck-mysekai":
			err = testDeckRecommendAuto(env.deckController, res.Query)
		default:
			err = testDeckRecommendAuto(env.deckController, res.Query)
		}
	case service.ModuleSK:
		return fmt.Errorf("sk module requires JSON input file, cannot be run from auto parsing alone")
	case service.ModuleMysekai:
		return fmt.Errorf("mysekai module requires JSON input file, cannot be run from auto parsing alone")
	default:
		return fmt.Errorf("cannot execute resolved command directly: %v", res)
	}

	if err != nil {
		slog.Error("Module execution failed", "module", res.Module, "mode", res.Mode, "error", err)
		return err
	}
	return nil
}

func (env *cliEnv) ensureCommand(mode, cmd string) (string, error) {
	if strings.TrimSpace(cmd) != "" {
		return cmd, nil
	}
	return env.defaultCommand(mode)
}

func (env *cliEnv) defaultCommand(mode string) (string, error) {
	switch strings.ToLower(mode) {
	case "card-detail":
		return "/闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閹冣挃闁硅櫕鎹囬垾鏃堝礃椤忎礁浜鹃柨婵嗙凹缁ㄥジ鏌熼惂鍝ョМ闁哄矉缍侀、姗€鎮欓幖顓燁棧闂備線娼уΛ娆戞暜閹烘缍栨繝闈涱儐閺呮煡鏌涘☉鍗炲妞ゃ儲鑹鹃埞鎴炲箠闁稿﹥顨嗛幈銊╂倻閽樺锛涢梺缁樺姉閸庛倝宕戠€ｎ喗鐓熸俊顖濆吹濠€浠嬫煃瑜滈崗娑氭濮橆剦鍤曢柟缁㈠枛椤懘鏌ｅΟ鑽ゅ灩闁搞儯鍔庨崢閬嶆⒑閺傘儲娅呴柛鐔村妽缁傛帡鏁傞崜褏锛滃銈嗘⒒閹虫挻鏅堕幓鎹涘酣宕惰闊剟鏌熼鐣屾噰妞ゃ垺妫冨畷鐔煎Ω閵夈倕顥氶梻浣告惈缁嬩線宕㈤懖鈺冪焼濠㈣泛鏈崰鎰扮叓閸ャ劍绀€闁搞劍绻傞埞鎴︽偐鐎圭姴顥濈紓浣哄缂嶄線寮婚敐鍛傜喖鎮滃Ο閿嬬亞婵犵妲呴崑鍛垝閹炬剚娼栭柧蹇撴贡閻捇鏌涢埄鍐炬當妞ゅ繐鐡ㄧ换娑㈠醇閻斿摜顦伴梺鍝勭焿缂嶄線鐛Ο灏栧亾濞戞瑯鐒芥い锔哄妽缁绘盯鏁愰崨顔芥倷闂佹寧娲︽禍婵囩┍婵犲洤閱囬柡鍥╁仜缁愭稑顪冮妶鍡樺暗闁哥姵鍔欓、娆撳礋椤愮喐鏂€闂佺粯鍔曞Ο濠囧吹閻斿皝鏀芥い鏃囧亹鏁堥悗瑙勬礀閻栧吋淇婂宀婃Щ濡?190", nil
	case "card-box":
		return "/闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閹冣挃闁硅櫕鎹囬垾鏃堝礃椤忎礁浜鹃柨婵嗙凹缁ㄥジ鏌熼惂鍝ョМ闁哄矉缍侀、姗€鎮欓幖顓燁棧闂備線娼уΛ娆戞暜閹烘缍栨繝闈涱儐閺呮煡鏌涘☉鍗炲妞ゃ儲鑹鹃埞鎴炲箠闁稿﹥顨嗛幈銊╂倻閽樺锛涢梺缁樺姉閸庛倝宕戠€ｎ喗鐓熸俊顖濆吹濠€浠嬫煃瑜滈崗娑氭濮橆剦鍤曢柟缁㈠枛椤懘鏌ｅΟ鑽ゅ灩闁搞儯鍔庨崢閬嶆⒑閺傘儲娅呴柛鐔村妽缁傛帡鏁傞崜褏锛滃銈嗘⒒閹虫挻鏅堕幓鎹涘酣宕惰闊剟鏌熼鐣屾噰妞ゃ垺妫冨畷鐔煎Ω閵夈倕顥氶梻浣告惈缁嬩線宕㈤懖鈺冪焼濠㈣泛鏈崰鎰扮叓閸ャ劍绀€闁搞劍绻傞埞鎴︽偐鐎圭姴顥濈紓浣哄缂嶄線寮婚敐鍛傜喖鎮滃Ο閿嬬亞婵犵妲呴崑鍛垝閹炬剚娼栭柧蹇撴贡閻捇鏌涢埄鍐炬當妞ゅ繐鐡ㄧ换娑㈠醇閻斿摜顦伴梺鍝勭焿缂嶄線鐛Ο灏栧亾濞戞瑯鐒芥い锔哄妽缁绘盯鏁愰崨顔芥倷闂佹寧娲︽禍婵囩┍婵犲洤閱囬柡鍥╁仜缁愭稑顪冮妶鍡樺暗闁哥姵鍔欓、娆撳礋椤愮喐鏂€闂佺粯鍔曞Ο濠囧吹閻斿皝鏀芥い鏃囧亹鏁堥悗瑙勬礀閻栧吋淇婂宀婃Щ濡?mnr", nil
	case "music", "music-detail":
		return "/闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閹冣挃闁硅櫕鎹囬垾鏃堝礃椤忎礁浜鹃柨婵嗙凹缁ㄥジ鏌熼惂鍝ョМ闁哄矉缍侀、姗€鎮欓幖顓燁棧闂備線娼уΛ娆戞暜閹烘缍栨繝闈涱儐閺呮煡鏌涘☉鍗炲妞ゃ儲鑹鹃埞鎴炲箠闁稿﹥顨嗛幈銊╂倻閽樺锛涢梺缁樺姉閸庛倝宕戠€ｎ喗鐓熸俊顖濆吹濠€浠嬫煃瑜滈崗娑氭濮橆剦鍤曢柟缁㈠枛椤懘鏌ｅΟ鑽ゅ灩闁搞儯鍔庨崢閬嶆⒑閺傘儲娅呴柛鐔村妽缁傛帡鏁傞崜褏锛滃銈嗘⒒閹虫挻鏅堕幓鎹涘酣宕惰闊剟鏌熼鐣屾噰妞ゃ垺妫冨畷鐔煎Ω閵夈倕顥氶梻浣告惈缁嬩線宕㈤懖鈺冪焼濠㈣泛鏈崰鎰扮叓閸ャ劍绀€闁搞劍绻傞埞鎴︽偐鐎圭姴顥濈紓浣哄缂嶄線寮婚敐鍛傜喖鎮滃Ο閿嬬亞婵犵妲呴崑鍛垝閹炬剚娼栭柧蹇撴贡閻捇鏌涢埄鍐炬當妞ゅ繐鐡ㄧ换娑㈠醇閻斿摜顦伴梺鍝勭焿缂嶄線鐛Ο灏栧亾闂堟稒鍟為柛锝庡枤缁辨挻鎷呮禒瀣懙闂佸湱顭堥…鐑界嵁韫囨稑宸濇い鏃囨瀵潡姊虹憴鍕剹闁告娅ｉ懞閬嶆嚍閵壯咁啎闂佺懓顕慨鐢稿汲椤掑嫭鐓曢柍瑙勫劤娴?1", nil
	case "music-brief":
		return "master:1,2,3", nil
	case "music-list":
		return "/婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻娑樷槈濮楀牊鏁鹃梺鍛婄懃缁绘﹢寮婚敐澶婄闁挎繂妫Λ鍕⒑閸濆嫷鍎庣紒鑸靛哺瀵鎮㈤崗灏栨嫽闁诲酣娼ф竟濠偽ｉ鍓х＜闁绘劦鍓欓崝銈嗐亜椤撶姴鍘寸€殿喖顭烽弫鎰緞婵犲嫮鏉告俊鐐€栫敮濠囨倿閿曞倸纾块柟鍓х帛閳锋垿鏌熼懖鈺佷粶濠碘€炽偢閺屾稒绻濋崒娑樹淮濡ょ姷鍋涢崯浼村箟閹绢喖绀嬫い鎺嗗亾妞ゎ偓绻濆娲偡閺夋寧些闂佹椿鍘奸悧鎾崇暦濠靛宸濇い鎾寸⊕閺傗偓闂備胶绮崝鏍亹閸愵喖绠栭柟杈鹃檮閻撶喐淇婇妶鍕厡闁活厽甯楅〃銉╂倷瀹割喗鈻堝Δ鐘靛仦閻楁骞忛崨瀛樺仭濡鑳剁紞宥夋⒒閸屾瑧顦﹀鐟帮躬瀹曟垿宕ㄩ娑樺簥闂佸憡娲﹂崹閬嶅磻閿熺姵鐓忓┑鐐戝啯鍣介柨娑欑箖缁绘稒娼忛崜褏蓱閻熸粍婢橀崯顖滅矉瀹ュ鍊锋い鎺戝€婚惁鍫ユ⒑闂堟盯鐛滅紒韫矙閹繝骞囬悧鍫㈠幈闂佸搫鍟犻崑鎾绘煟濡ゅ啫鈻堟鐐插暣閸ㄩ箖寮妷锔绘綌婵犳鍠楅敃顐ゅ緤閼恒儳顩查柣鎰靛墻濞堜粙鏌ｉ幇顖氱毢濞寸姰鍨介弻娑㈠籍閳ь剛鍠婂鍥ㄥ床婵炴垯鍨圭粻缁樸亜閺冨洤袚闁哄棛濮风槐鎾存媴缁嬪簱鍋撻崫銉х煋闁割偅娲栫粻鐔兼煙缂併垹鏋涚紒鈧€ｎ偁浜滈柟鎹愭硾椤庡矂鏌涢悢鐑藉弰闁哄矉绲鹃幆鏃堟晲閸℃鐣┑鐘灮閹虫捇鏁冮鍕殾濞村吋娼欓崡鎶芥煟濡儤鎯堢亸蹇涙⒒娴ｅ搫甯舵い顐ｆ礋瀹曟劕螖閸愵煈妫滄繝闈涘€搁幉锟犳偂閻旈晲绻嗘い鏍ㄧ箖椤忕姴霉閻樺磭娲撮柡灞剧洴婵″爼宕掑顐㈩棜闂傚倸鍊搁崐鎼佸磹缁嬫５娲Χ婢癸箑娲幃鐣岀矙閼愁垱鎲伴梻渚€娼ч…鍫ュ磿椤曗偓瀵劍绂掔€ｎ偆鍘遍梺鏂ユ櫅閸橀箖顢旈崟銊︾€洪梺闈涚墕椤︿即鍩涢幋锔藉仭婵炲棗绻愰鈺冪磼閹绘帩鐓奸柡宀€鍠撻幏鐘侯槾缂佲檧鍋撳┑鐘殿暜缁辨洟寮拠鑼殾闁绘梻鈷堥弫宥嗙箾閹寸偟鎳勯柣婵囨礋濮婃椽鎳￠妶鍛呫垺绻涘ù瀣珔妞ゆ洩缍侀獮姗€骞嗛弶鍟冣剝绻濋悽闈浶ｇ痪鏉跨Ч閸╂盯骞嬮敂钘夆偓鐢告煕閿旇骞栨い搴℃湰缁绘盯宕楅悡搴☆潚闂佸搫鏈惄顖涗繆閹间礁唯鐟滃秹鎮￠悢鍏尖拺缂備焦眉缁堕亶鏌涢悩鎰佹畼闁瑰箍鍨归埞鎴犫偓锝庝簽閸婄偤姊洪棃娴ゆ盯宕橀妸褏鏉芥繝鐢靛Х閺佸憡鎱ㄩ悜钘夋瀬闁告稑锕ラ崣蹇涙煙缂併垹鏋涢柣銈庡枟閵囧嫰寮介妸褏鐓傞梺娲诲幖濡繈寮诲☉銏╂晝闁绘ɑ褰冩慨鏇㈡⒑缂佹ɑ灏柛鐔跺嵆楠炲绮欏▎鍓у弳闂佸壊鍋呯换鍕囬鐐╂斀妞ゆ梻鐡旈悞鐐箾婢跺鈯曢悗闈涖偢閹晝鎷犻懠顒夊晣婵犵數鍎戠紞鍡涘礈濞嗘劒绻嗛柤娴嬫櫇绾惧ジ鏌嶈閸撴艾顕ラ崟顖氱疀妞ゆ挾濮撮獮宥夋⒑閸濆嫷妲搁柣妤€妫欓弲鑸电鐎ｎ偄浠奸梻渚囧墮缁夌敻鎮″☉銏＄厱闁哄洦顨嗗▍鍛存煕閺冩挾鐣甸柡?ma 32", nil
	case "music-progress":
		return "/闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤濠€閬嶅焵椤掑倹鍤€閻庢凹鍙冨畷宕囧鐎ｃ劋姹楅梺鍦劋閸ㄥ綊宕愰悙鐑樺仭婵犲﹤鍟扮粻鑽も偓娈垮枟婵炲﹪寮崘顔肩＜婵炴垶鑹鹃獮鍫熶繆閻愵亜鈧倝宕㈡禒瀣瀭闁割煈鍋嗛々鍙夌節闂堟侗鍎愰柣鎾存礃缁绘盯宕卞Δ鍐唺缂備胶濮撮…鐑藉蓟閳ュ磭鏆嗛柍褜鍓熷畷浼村箻閼告娼熼梺鍦劋椤ㄥ懘锝為崨瀛樼厽婵☆垵娅ｉ敍宥囨喐闁箑鐏︽慨濠勭帛閹峰懘鎳為妷锝傚亾閸愵亞纾奸柍褜鍓氶幏鍛偘閳ュ厖澹曞┑顔筋焽閸樠勬櫠閹绢喗鐓冮柕澶樺灠椤╊剟鏌熼悷鏉款伃濠碘剝鎮傞弫鍐焵椤掑倸顥氭い鏍ㄧ〒缁犻箖鏌熼悙顒佺稇闁搞値鍓熼弻娑㈠Ω閵堝懎绁悗瑙勬礃缁挸鐣峰Ο渚晠妞ゆ棁妫勯惁婊堟⒒娓氣偓濞佳囨偋閸℃稑纾归柣鐔稿閺嬪秹鏌涢埄鍐姇闁绘挾鍠栭弻鐔煎垂椤旂⒈浼€缂備讲妾ч崑鎾翠繆閵堝洤啸闁稿鐩畷顖涘閺夋垹鐤呴梺鍦檸閸犳牜绮堢€ｎ偁浜滈柟鍝勬娴滄儳顪冮妶搴′壕缂佺粯绻傞～蹇撁洪鍛画闂佸搫顦扮€笛傜昂闂傚倷鐒﹀鍧楀储婵傚憡鍋嬮柟鎯х－閺嗭附绻濋棃娑卞剰鐎瑰憡绻堥幃妤€鈽夊▍杈ㄥ哺楠炲繘鎼归崷顓狅紳闂佺鏈悷褔宕濆澶嬬叆婵浜壕鍏间繆閵堝倸浜鹃梺绋匡龚瀹曢潧危閹版澘绠虫俊銈咃攻閺呪晠姊烘导娆戝埌闁哄牜鍓熷畷铏鐎涙ê鈧灚绻涢崼婵堜虎闁哄绋掗妵鍕敇閻樻彃骞嬮悗娈垮櫘閸嬪﹪鐛崶顒€绾ч柛顭戝枤閻涒晠姊绘担渚劸闁哄牜鍓涚划娆撳箳濡炴繂顦靛畷濂稿Ψ閿旀儳骞堥梻浣哥枃椤宕曢搹顐ゎ洸闁绘劦鍏涚换鍡涙煟閹板吀绨婚柍褜鍓氶悧婊堝极椤曗偓楠炴帡寮崫鍕濠殿喗顭囬崢褍顕ｉ閿亾鐟欏嫭绀冮柛銊ョ秺閸┾偓妞ゆ帒锕︾粔闈浢瑰鍕⒌鐎殿喓鍔嶇粋鎺斺偓锝庡亞閸樿棄鈹戦埥鍡楃仴妞ゆ泦鍛筏濠电姵纰嶉悡娑㈡煕閳╁啰鎳冮柡瀣灴閺屾洟宕惰椤忣參鏌涢埡瀣瘈鐎规洘甯掗…銊╁礃閵娧冩憢闂傚倸鍊风粈渚€骞栭锕€瀚夋い鎺嗗亾閻撱倝鏌ｉ弮鍫闁哄棴绠撻弻锝夊箻瀹曞洤鍝洪梺琛″亾濞寸姴顑嗛悡鏇㈡煏婢跺牆濡奸柣鎾村姍閺屻劑鎮㈤弶鎴濆Б缂備浇椴搁幑鍥х暦閹烘垟鏋庨柟鐑樺灥鐢垰鈹戦悩鍨毄闁稿鍔欏畷鎴﹀箻閸撲胶锛濇繛杈剧到閹碱偅鐗庨梻浣虹帛椤ㄥ牊鎱ㄩ幘顕呮晪闁挎繂妫涚弧鈧┑顔斤供閸樺吋绂嶅鍫熺厽闁绘劕顕埢鎾绘煃瑜滈崜娆撳储閺嶃劎鐝堕柡鍥ュ灪閳锋垶鎱ㄩ悷鐗堟悙闁诲繆鏅犻弻鐔访圭€Ｑ冧壕闁哄倶鍎查悗鍝勨攽閻樿宸ラ柟铏姉婢规洘绺界粙璺ㄩ獓闂佸壊鍋掗崑鍕濠靛鐓熸繝鍨姃闁垶鏌＄仦鐐鐎规洘鍎奸ˇ鍙夈亜韫囷絽澧扮紒杈ㄥ浮閹晛鐣烽崶銊ュ灡婵°倗濮烽崑鐐衡€﹂崶顒€绠查柛鏇ㄥ幗閸嬫﹢鏌嶉埡浣告殭闁绘繈浜跺?ma", nil
	case "music-chart":
		return "/闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤濠€閬嶅焵椤掑倹鍤€閻庢凹鍙冨畷宕囧鐎ｃ劋姹楅梺鍦劋閸ㄥ綊宕愰悙鐑樺仭婵犲﹤鍟扮粻鑽も偓娈垮枟婵炲﹪寮崘顔肩＜婵炴垶鑹鹃獮鍫熶繆閻愵亜鈧倝宕㈡禒瀣瀭闁割煈鍋嗛々鍙夌節闂堟侗鍎愰柣鎾存礃缁绘盯宕卞Δ鍐唺缂備胶濮撮…鐑藉蓟閳ュ磭鏆嗛柍褜鍓熷畷浼村箻閼告娼熼梺鍦劋椤ㄥ懘锝為崨瀛樼厽婵☆垵娅ｉ敍宥囨喐闁箑鐏︽慨濠勭帛閹峰懘鎳為妷锝傚亾閸愵亞纾奸柍褜鍓氶幏鍛偘閳ュ厖澹曞┑顔筋焽閸樠勬櫠閹绢喗鐓冮柕澶樺灠椤╊剟鏌熼悷鏉款伃濠碘剝鎮傞弫鍐焵椤掑倸顥氭い鏍ㄧ〒缁犻箖鏌熼悙顒佺稇闁搞値鍓熼弻娑㈠Ω閵堝懎绁悗瑙勬礃缁挸鐣峰Ο渚晠妞ゆ棁妫勯惁婊堟⒒娓氣偓濞佳囨偋閸℃稑纾归柣鐔稿閺嬪秹鏌涢埄鍐姇闁绘挾鍠栭弻鐔煎垂椤旂⒈浼€缂備讲妾ч崑鎾翠繆閵堝洤啸闁稿鐩畷顖涘閺夋垹鐤呴梺鍦檸閸犳牜绮堢€ｎ偁浜滈柟鍝勬娴滄儳顪冮妶搴′壕缂佺粯绻傞～蹇撁洪鍛画闂佸搫顦扮€笛傜昂闂傚倷鐒﹀鍧楀储婵傚憡鍋嬮柟鎯х－閺嗭附绻濋棃娑卞剰鐎瑰憡绻堥幃妤€鈽夊▍杈ㄥ哺楠炲繘鎼归崷顓狅紳闂佺鏈悷褔宕濆澶嬬叆婵浜壕鍏间繆閵堝倸浜鹃梺绋匡龚瀹曢潧危閹版澘绠虫俊銈咃攻閺呪晠姊烘导娆戝埌闁哄牜鍓熷畷铏鐎涙ê鈧灚绻涢崼婵堜虎闁哄绋掗妵鍕敇閻樻彃骞嬮悗娈垮櫘閸嬪﹪鐛崶顒€绾ч柛顭戝枤閻涒晠姊绘担渚劸闁哄牜鍓涚划娆撳箳濡炴繂顦靛畷濂稿Ψ閿旀儳骞堥梻浣哥枃椤宕曢搹顐ゎ洸闁绘劦鍏涚换鍡涙煟閹板吀绨婚柍褜鍓氶悧婊堝极椤曗偓楠炴帡寮崫鍕濠殿喗顭囬崢褍顕ｉ閿亾鐟欏嫭绀冮柛銊ョ秺閸┾偓妞ゆ帒锕︾粔鐢告煕閻樻剚娈滈柟顕嗙節閹垽宕楅懖鈺佸箥闂傚倷绶￠崣蹇曠不閹寸偞娅犻柡澶嬶紩瑜版帗鍋愮€规洖娲犳慨鍥⒑闂堟稒鎼愰悗姘嵆閵嗕線寮撮姀鈩冩珳闂佺硶鍓濋悷顖毼ｈぐ鎺撯拻濞达絽鎲￠幆鍫ユ偠濮樼厧浜扮€规洘绻傞悾婵嬪礋椤愩倕骞嬮梻浣侯攰閹活亞绮婚幋锔惧祦闁靛繈鍊栭悡鍐煕濠靛棗顏╅柍褜鍓氶崝姗€藝鐟欏嫭鍙忓┑鐘叉噺椤忕姷绱掓潏銊ョ瑨閾伙綁鎮归崶銊у弨闁诲海鍎ょ换婵嬫偨闂堟稐鍝楅柣蹇撶箲閻熲晛鐣锋导鏉戝唨鐟滄粓宕甸弴鐐╂斀闁绘ê纾。鏌ユ煛閸涱喗鍊愰柡宀嬬到铻ｉ柛顭戝枤濮ｃ垽姊虹粙鍖″姛闁稿鍊曢～蹇撁洪鍜佹濠电偞鍨兼ご姝屽€撮梻鍌欒兌椤牓鏌婇敐鍡欘洸闁割偅娲栭悘鎶芥煛瀹ュ骸骞栭梺鍗炴处缁绘繈妫冨☉妯绘闂佸搫鍊甸崑鎾绘⒒閸屾瑨鍏岀紒顕呭灦瀹曟繈寮撮悜鍡楁闂佸壊鍋呭ú鏍⒒椤栨稓绠剧€瑰壊鍠曠花濂告倵濮橆剚鍤囩€殿喖鐖煎畷鐓庮潩椤撶喓褰囩紓?1 ma", nil
	case "gacha-list":
		return "/闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閹冣挃闁硅櫕鎹囬垾鏃堝礃椤忎礁浜鹃柨婵嗙凹缁ㄥジ鏌熼惂鍝ョМ闁哄矉缍侀、姗€鎮欓幖顓燁棧闂備線娼уΛ娆戞暜閹烘缍栨繝闈涱儐閺呮煡鏌涘☉鍗炲妞ゃ儲鑹鹃埞鎴炲箠闁稿﹥顨嗛幈銊╂倻閽樺锛涘┑鐐村灍閹崇偤宕堕浣镐缓缂備礁顑嗙€笛囨倵椤掑嫭鈷戦柣鐔告緲閳锋梻绱掗鍛仸鐎规洘鍨块獮鍥偋閸垹骞堟繝鐢靛仜濡鎹㈤幇顓狀洸濞寸厧鐡ㄩ埛鎺楁煕閺囥劌浜滄い蹇ｅ亰閺岀喖鐛崹顔句患闂佸疇妫勯ˇ鍨叏閳ь剟鏌ｅΟ娲诲晱闁告艾鎳忕换婵嬫偨闂堟稐绮跺┑鈽嗗亝椤ㄥ牓骞戦姀銈呯闁归箖顤傚ù鍕⒑閸忚偐銈撮柡鍛箘婢规洟鎸婃竟婵嗙秺閺佹劙宕卞Δ鍐偡濠电偛鐡ㄧ划鎾剁不閺嵮屾綎婵炲樊浜滅粈鍫ユ煙缂佹ê绗傜紒銊ょ矙濮婃椽宕妷銉愶綁鏌熼崨濠傚姢妞ゆ洩缍€椤﹀綊鏌涢埞鎯т壕婵＄偑鍊栫敮鎺斺偓姘煎弮閸╂盯骞掑Δ浣哄幈闁诲繒鍋熼崑鎾绘儍閹存繍鐔嗙憸搴ｇ矙閹达箑违闁圭儤鍩堝鈺傘亜閹炬瀚弶褰掓煟鎼淬値娼愭繛鍙壝叅闁绘棁鍋愬畵渚€鏌涢妷顔煎闁稿顑夐弻娑㈩敃閻樿尙浠煎Δ鐘靛仜閻楁挸顫忓ú顏呭仭闁哄瀵ч崐顖炴⒑娴兼瑧绉靛ù婊庝簻閻ｇ柉銇愰幒鎾村劒闂備緡鍋呯粙鎾诲礈閵娾晜鈷戦悷娆忓閸熷繘鏌涢悩铏殤濠㈣娲熼幊鏍煛閸愵亷绱冲┑鐐舵彧缂嶁偓妞ゆ洘鐗犻幃鐐烘偩瀹€鈧壕濂告煟濡櫣锛嶅褑浜槐鎺撴綇閵娿儳顑傜紓浣介哺鐢帟鐏掗梺鍏肩ゴ閺呮繈寮舵禒瀣厽閹兼番鍊ゅ鎰箾閸欏鐒介柡渚囧櫍楠炴帒顪冮悜鈺佷壕闁挎洖鍊告儫闂佸疇妗ㄩ懗鍫曀囬妸鈺傗拺缂備焦锚婵箓鏌涢幘鏉戝摵闁诡垯绶氶獮鎺楀籍閸屾粣绱抽梻浣侯焾閺堫剟鎳濇ィ鍐ㄧ劦妞ゆ帊鐒﹂崐鎰偓瑙勬礃閸旀牠藝閻楀牊鍎熼柕蹇曞閳ь剚鐩娲传閸曨噮娼堕梺绋挎唉娴滎剚绔熼弴鐔虹瘈婵﹩鍘鹃崢鐢告⒑閸涘﹥瀵欓柛娑卞幘椤愬ジ姊绘担铏瑰笡闁规瓕顕х叅闁绘梻鍘ч拑鐔衡偓骞垮劚椤︻垶鎮″☉妯忓綊鏁愰崶鍓佸姼缂備胶濮撮…宄邦潖濞差亜绀傞柤娴嬫杺閸嬬偤姊洪崫鍕櫤缂侇喖鐭傞幃楣冩倻閽樺鎽曢梺闈涱檧婵″洭宕㈤柆宥嗏拺闁哄倶鍎插▍鍛存煕閻旇泛宓嗛柛鈺侊躬瀵挳濮€閿涘嫬骞楅梻浣筋潐閸庡磭澹曢鐘典笉濠电姵纰嶉悡娑㈡倶閻愭彃鈷旀繛鎻掔摠閵囧嫰濮€閳╁啰顦伴梺杞扮劍閸旀瑥鐣烽崼鏇炵厸闁告劏鏅滈惁鎺楁⒒閸屾艾鈧娆㈠顒夌劷闁煎鍊楃粈濠囨煃瑜滈崜姘跺Φ閸曨垼鏁冮柕蹇婃櫆閳诲牊绻濈喊澶岀？闁轰浇顕ч悾鐑芥偄绾拌鲸鏅╅梺鍛婃寙閸曨剛甯嗛梻鍌欐祰椤曆囧礄閻ｅ瞼绀婇柛鈩冾焽椤╁弶绻濇繝鍌氼仾闁告瑥绻橀弻銊╁即閻愭祴鍋撹ぐ鎺戠厱?p1", nil
	case "gacha-detail":
		latestID, err := env.pickLatestGachaID()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("/闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閹冣挃闁硅櫕鎹囬垾鏃堝礃椤忎礁浜鹃柨婵嗙凹缁ㄥジ鏌熼惂鍝ョМ闁哄矉缍侀、姗€鎮欓幖顓燁棧闂備線娼уΛ娆戞暜閹烘缍栨繝闈涱儐閺呮煡鏌涘☉鍗炲妞ゃ儲鑹鹃埞鎴炲箠闁稿﹥顨嗛幈銊╂倻閽樺锛涘┑鐐村灍閹崇偤宕堕浣镐缓缂備礁顑嗙€笛囨倵椤掑嫭鈷戦柣鐔告緲閳锋梻绱掗鍛仸鐎规洘鍨块獮鍥偋閸垹骞堟繝鐢靛仜濡鎹㈤幇顓狀洸濞寸厧鐡ㄩ埛鎺楁煕閺囥劌浜滄い蹇ｅ亰閺岀喖鐛崹顔句患闂佸疇妫勯ˇ鍨叏閳ь剟鏌ｅΟ娲诲晱闁告艾鎳忕换婵嬫偨闂堟稐绮跺┑鈽嗗亝椤ㄥ牓骞戦姀銈呯闁归箖顤傚ù鍕⒑閸忚偐銈撮柡鍛箘婢规洟鎸婃竟婵嗙秺閺佹劙宕卞Δ鍐偡濠电偛鐡ㄧ划鎾剁不閺嵮屾綎婵炲樊浜滅粈鍫ユ煙缂佹ê绗傜紒銊ょ矙濮婃椽宕妷銉愶綁鏌熼崨濠傚姢妞ゆ洩缍€椤﹀綊鏌涢埞鎯т壕婵＄偑鍊栫敮鎺斺偓姘煎弮閸╂盯骞掑Δ浣哄幈闁诲繒鍋熼崑鎾绘儍閹存繍鐔嗙憸搴ｇ矙閹达箑违闁圭儤鍩堝鈺傘亜閹炬瀚弶褰掓煟鎼淬値娼愭繛鍙壝叅闁绘棁鍋愬畵渚€鏌涢妷顔煎闁稿顑夐弻娑㈩敃閻樿尙浠煎Δ?%d", latestID), nil
	case "event-detail":
		return "/濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴ｆ閺嬩線鏌涘☉姗堟敾闁告瑥绻橀弻锝夊箣閿濆棭妫勯梺鍝勵儎缁舵岸寮诲☉妯锋婵鐗婇弫楣冩⒑閸涘﹦鎳冪紒缁橈耿瀵鏁愭径濠勵吅闂佹寧绻傚Λ顓炍涢崟顖涒拺闁告繂瀚烽崕搴ｇ磼閼搁潧鍝虹€殿喖顭烽幃銏ゅ礂鐏忔牗瀚介梺璇查叄濞佳勭珶婵犲伣锝夘敊閸撗咃紲闂佺粯鍔﹂崜娆撳礉閵堝棎浜滄い鎾跺Т閸樺鈧鍠栭…鐑藉极閹版澘宸濋柛灞剧矊閺嬫稓鈧娲﹂崑濠傜暦閻旂⒈鏁嗛柍褜鍓熼崺鈧い鎺嗗亾闂佸府绲介～蹇涙惞閸︻厾锛滃┑鈽嗗灥濞咃綁鎮烽妸鈺傗拺闁圭瀛╃壕鎼佹煕鎼达紕浠涘ǎ鍥э躬閹晫绮欑捄顭戞Ч婵＄偑鍊栭弻銊︽櫠娴犲绾ч柟闂寸劍閳锋垿鎮峰▎蹇擃仼缂佲偓閸愨晝绠惧璺侯儑濞叉挳鏌涢埡鍌滄创鐎规洖銈告俊鐑芥晜鐟欏嫬顏归梻浣告惈椤﹂亶宕戦悙瀵哥彾闁糕剝铔嬮崶顒佹櫆濠殿喗鍔掔花濠氭⒑閻熸壆鎽犵紒璇插暣瀹曟澘螣濞茬粯顔旈梺缁樺姇濡﹤螣閳ь剟鎮楃憴鍕８闁告柨绉堕幑銏犫攽鐎ｎ亞顦ㄩ梺闈涱焾閸庨亶鐓㈤梻鍌氬€搁崐椋庢濮橆剦鐒界憸鏃堝灳閿曞倸閱囬柕澶堝劤閿涚喖姊虹憴鍕姸濠殿喓鍊濆畷?current", nil
	case "event-list":
		return "/events wl", nil
	case "education-challenge":
		return "/挑战信息", nil
	case "profile":
		return "1", nil
	case "stamp-list":
		return "D:/github/testfile/stamp_list.json", nil
	case "misc-chara-birthday":
		return "D:/github/testfile/misc_birthday.json", nil
	case "score-control":
		return "D:/github/testfile/score_control.json", nil
	case "score-custom-room":
		return "D:/github/testfile/score_custom_room.json", nil
	case "score-music-meta":
		return "D:/github/testfile/score_music_meta.json", nil
	case "score-music-board":
		return "D:/github/testfile/score_music_board.json", nil
	case "deck-recommend":
		return "D:/github/testfile/deck_recommend.json", nil
	case "deck-recommend-auto":
		return "D:/github/testfile/deck_recommend.json", nil
	case "sk-line":
		return "D:/github/testfile/sk_line.json", nil
	case "sk-query":
		return "D:/github/testfile/sk_query.json", nil
	case "sk-check-room":
		return "D:/github/testfile/sk_check_room.json", nil
	case "sk-speed":
		return "D:/github/testfile/sk_speed.json", nil
	case "sk-player-trace":
		return "D:/github/testfile/sk_player_trace.json", nil
	case "sk-rank-trace":
		return "D:/github/testfile/sk_rank_trace.json", nil
	case "sk-winrate":
		return "D:/github/testfile/sk_winrate.json", nil
	case "mysekai-resource":
		return "D:/github/testfile/mysekai_resource.json", nil
	case "mysekai-fixture-list":
		return "D:/github/testfile/mysekai_fixture_list.json", nil
	case "mysekai-fixture-detail":
		return "D:/github/testfile/mysekai_fixture_detail.json", nil
	case "mysekai-door-upgrade":
		return "D:/github/testfile/mysekai_door_upgrade.json", nil
	case "mysekai-music-record":
		return "D:/github/testfile/mysekai_music_record.json", nil
	case "mysekai-talk-list":
		return "D:/github/testfile/mysekai_talk_list.json", nil
	default:
		return "", fmt.Errorf("mode %s requires -cmd", mode)
	}
}

func (env *cliEnv) pickLatestGachaID() (int, error) {
	gachas := env.masterdata.GetGachas()
	if len(gachas) == 0 {
		return 0, fmt.Errorf("no gacha data available")
	}
	latest := gachas[0]
	for _, g := range gachas {
		if g.StartAt > latest.StartAt {
			latest = g
		}
	}
	return latest.ID, nil
}

func preprocessCommand(cmd string, keywords ...string) string {
	raw := strings.TrimSpace(strings.TrimPrefix(cmd, "/"))
	for _, kw := range keywords {
		if kw == "" {
			continue
		}
		raw = strings.ReplaceAll(raw, kw, "")
	}
	return strings.TrimSpace(raw)
}

func stripLeadingRegionToken(raw string) string {
	_, rest := extractLeadingRegionToken(raw)
	return rest
}

func extractLeadingRegionToken(raw string) (string, string) {
	parts := strings.Fields(strings.TrimSpace(raw))
	if len(parts) == 0 {
		return "", ""
	}
	regionSet := map[string]struct{}{
		"jp": {}, "en": {}, "cn": {}, "tw": {}, "kr": {},
	}
	first := strings.ToLower(strings.TrimSpace(parts[0]))
	if _, ok := regionSet[first]; ok {
		return first, strings.TrimSpace(strings.Join(parts[1:], " "))
	}
	return "", strings.TrimSpace(strings.Join(parts, " "))
}

func testMusicDetail(ctrl *controller.MusicController, svc *service.MusicSearchService, cmd string) error {
	raw := preprocessCommand(cmd, "/查曲", "查曲", "查歌", "查乐", "查询乐曲", "查音乐")
	region, cleaned := extractLeadingRegionToken(raw)
	if region == "" {
		region = "jp"
	}
	if svc != nil {
		if music, err := svc.Search(cleaned); err == nil {
			cleaned = strconv.Itoa(music.ID)
		}
	}
	query := model.MusicQuery{Query: cleaned, UserID: "test_user", Region: region}

	start := time.Now()
	imageData, err := ctrl.RenderMusicDetail(query)
	if err != nil {
		return fmt.Errorf("render music failed: %w", err)
	}
	fmt.Printf("Render music success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("music_detail", 0, imageData)
}

func testMusicBriefList(ctrl *controller.MusicController, cmd string) error {
	raw := strings.TrimSpace(strings.TrimPrefix(cmd, "/"))
	diff := "master"
	region := "jp"
	parsedRegion, payload := extractLeadingRegionToken(raw)
	if parsedRegion != "" {
		region = parsedRegion
	}
	if strings.Contains(raw, ":") {
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) == 2 {
			if strings.TrimSpace(parts[0]) != "" {
				diff = strings.TrimSpace(parts[0])
			}
			payload = strings.TrimSpace(parts[1])
			parsedRegion, trimmed := extractLeadingRegionToken(payload)
			if parsedRegion != "" {
				region = parsedRegion
			}
			payload = trimmed
		}
	}

	tokens := strings.FieldsFunc(payload, func(r rune) bool {
		return r == ',' || r == ';' || r == ' '
	})
	var ids []int
	for _, t := range tokens {
		if t == "" {
			continue
		}
		id, err := strconv.Atoi(t)
		if err != nil {
			fmt.Printf("Skip invalid id %s\n", t)
			continue
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return fmt.Errorf("no valid music IDs to render")
	}

	fmt.Printf("Testing Music Brief List: diff=%s, ids=%v\n", diff, ids)
	start := time.Now()
	imageData, err := ctrl.RenderMusicBriefList(ids, diff, region)
	if err != nil {
		return fmt.Errorf("render music brief list failed: %w", err)
	}
	fmt.Printf("Render music brief list success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("music_brief_list", 0, imageData)
}

func testMusicList(ctrl *controller.MusicController, cmd string) error {
	query, err := parseMusicListCommand(cmd)
	if err != nil {
		return fmt.Errorf("parse music list command failed: %w", err)
	}

	start := time.Now()
	imageData, err := ctrl.RenderMusicList(query)
	if err != nil {
		return fmt.Errorf("render music list failed: %w", err)
	}

	fmt.Printf("Render music list success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("music_list", 0, imageData)
}

func testMusicProgress(ctrl *controller.MusicController, cmd string) error {
	query, err := parseMusicProgressCommand(cmd)
	if err != nil {
		return fmt.Errorf("parse music progress command failed: %w", err)
	}

	start := time.Now()
	imageData, err := ctrl.RenderMusicProgress(query)
	if err != nil {
		return fmt.Errorf("render music progress failed: %w", err)
	}

	fmt.Printf("Render music progress success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("music_progress", 0, imageData)
}

func testMusicChart(ctrl *controller.MusicController, cmd string) error {
	query, err := parseMusicChartCommand(cmd)
	if err != nil {
		return fmt.Errorf("parse music chart command failed: %w", err)
	}
	start := time.Now()
	imageData, err := ctrl.RenderMusicChart(query)
	if err != nil {
		return fmt.Errorf("render music chart failed: %w", err)
	}
	fmt.Printf("Render music chart success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("music_chart", 0, imageData)
}

func testMusicRewardsDetail(ctrl *controller.MusicController, cmd string) error {
	query, err := parseMusicRewardsDetailCommand(cmd)
	if err != nil {
		return fmt.Errorf("parse music rewards detail command failed: %w", err)
	}

	start := time.Now()
	imageData, err := ctrl.RenderMusicRewardsDetail(query)
	if err != nil {
		return fmt.Errorf("render music rewards detail failed: %w", err)
	}

	fmt.Printf("Render music rewards detail success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("music_rewards_detail", 0, imageData)
}

func testMusicRewardsBasic(ctrl *controller.MusicController, cmd string) error {
	query, err := parseMusicRewardsBasicCommand(cmd)
	if err != nil {
		return fmt.Errorf("parse music rewards basic command failed: %w", err)
	}

	start := time.Now()
	imageData, err := ctrl.RenderMusicRewardsBasic(query)
	if err != nil {
		return fmt.Errorf("render music rewards basic failed: %w", err)
	}

	fmt.Printf("Render music rewards basic success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("music_rewards_basic", 0, imageData)
}

func testGachaList(ctrl *controller.GachaController, cmd string) error {
	query := parseGachaListCommand(cmd)
	start := time.Now()
	imageData, err := ctrl.RenderGachaList(query)
	if err != nil {
		return fmt.Errorf("render gacha list failed: %w", err)
	}
	fmt.Printf("Render gacha list success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("gacha_list", query.Page, imageData)
}

func testGachaDetail(env *cliEnv, cmd string) error {
	query, err := parseGachaDetailCommand(cmd)
	if err != nil {
		return fmt.Errorf("parse gacha detail command failed: %w", err)
	}
	if query.GachaID < 0 && env.masterdata != nil {
		gachas := env.masterdata.GetGachas()
		if len(gachas) > 0 {
			if query.GachaID == -1 && query.NegIndex > 0 {
				idx := len(gachas) - query.NegIndex
				if idx >= 0 && idx < len(gachas) {
					query.GachaID = gachas[idx].ID
				}
			} else if query.GachaID == -2 && query.EventID > 0 {
				if event, err := env.masterdata.GetEventByID(query.EventID); err == nil && event != nil {
					// Find gacha that starts around the same time
					for _, g := range gachas {
						if strings.Contains(strings.ToLower(g.Name), "it's back") || strings.Contains(strings.ToLower(g.Name), "复刻") {
							continue
						}
						if getAbsDiff(g.StartAt, event.StartAt) < int64(time.Hour*48/time.Millisecond) {
							query.GachaID = g.ID
							break
						}
					}
				}
			}
		}
	}

	start := time.Now()
	imageData, err := env.gachaController.RenderGachaDetail(query)
	if err != nil {
		return fmt.Errorf("render gacha detail failed: %w", err)
	}
	fmt.Printf("Render gacha detail success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("gacha_detail", query.GachaID, imageData)
}

func getAbsDiff(a, b int64) int64 {
	if a > b {
		return a - b
	}
	return b - a
}

func testEventDetail(ctrl *controller.EventController, search *service.EventSearchService, cmd string) error {
	raw := preprocessEventCommand(cmd)
	region, cleaned := extractLeadingRegionToken(raw)
	if region == "" {
		region = "jp"
	}
	raw = cleaned
	if raw == "" {
		raw = "current"
	}
	event, err := search.Search(raw)
	if err != nil {
		return fmt.Errorf("failed to find event: %w", err)
	}
	query := model.EventDetailQuery{
		Region:  region,
		EventID: event.ID,
	}
	start := time.Now()
	imageData, err := ctrl.RenderEventDetail(context.Background(), query)
	if err != nil {
		return fmt.Errorf("render event detail failed: %w", err)
	}
	fmt.Printf("Render event detail success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("event_detail", event.ID, imageData)
}

func testEventList(ctrl *controller.EventController, parser *service.EventParser, cmd string) error {
	query, err := parseEventListCommand(cmd, parser)
	if err != nil {
		return fmt.Errorf("parse event list command failed: %w", err)
	}
	start := time.Now()
	imageData, err := ctrl.RenderEventList(query)
	if err != nil {
		return fmt.Errorf("render event list failed: %w", err)
	}
	fmt.Printf("Render event list success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("event_list", 0, imageData)
}

func testEventRecord(ctrl *controller.EventController, cmd string) error {
	raw := strings.TrimSpace(cmd)
	var req model.EventRecordRequest
	if err := loadQueryFromFile(raw, &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderEventRecord(req)
	if err != nil {
		return fmt.Errorf("render event record failed: %w", err)
	}
	fmt.Printf("Render event record success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("event_record", 0, imageData)
}

func testEducationChallengeLive(ctrl *controller.EducationController, cmd string) error {
	raw := strings.TrimSpace(cmd)
	start := time.Now()
	var (
		imageData []byte
		err       error
	)
	switch determineEducationInputType(raw) {
	case educationInputJSON:
		var req model.ChallengeLiveDetailsRequest
		if err := loadQueryFromFile(strings.TrimSpace(raw), &req); err != nil {
			return err
		}
		imageData, err = ctrl.RenderChallengeLiveDetail(req)
	case educationInputRegion:
		region := parseEducationRegion(raw)
		imageData, err = ctrl.RenderChallengeLiveDetailFromUser(region)
	default:
		imageData, err = ctrl.RenderChallengeLiveDetailFromUser("jp")
	}
	if err != nil {
		return fmt.Errorf("render education challenge failed: %w", err)
	}
	fmt.Printf("Render education challenge success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("education_challenge", 0, imageData)
}

func testEducationPowerBonus(ctrl *controller.EducationController, cmd string) error {
	var req model.PowerBonusDetailRequest
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderPowerBonusDetail(req)
	if err != nil {
		return fmt.Errorf("render education power bonus failed: %w", err)
	}
	fmt.Printf("Render education power bonus success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("education_power_bonus", 0, imageData)
}

func testEducationAreaItem(ctrl *controller.EducationController, cmd string) error {
	var req model.AreaItemUpgradeMaterialsRequest
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderAreaItemMaterials(req)
	if err != nil {
		return fmt.Errorf("render education area item failed: %w", err)
	}
	fmt.Printf("Render education area item success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("education_area_item", 0, imageData)
}

func testEducationBonds(ctrl *controller.EducationController, cmd string) error {
	var req model.BondsRequest
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderBonds(req)
	if err != nil {
		return fmt.Errorf("render education bonds failed: %w", err)
	}
	fmt.Printf("Render education bonds success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("education_bonds", 0, imageData)
}

func testEducationLeaderCount(ctrl *controller.EducationController, cmd string) error {
	var req model.LeaderCountRequest
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderLeaderCount(req)
	if err != nil {
		return fmt.Errorf("render education leader count failed: %w", err)
	}
	fmt.Printf("Render education leader count success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("education_leader_count", 0, imageData)
}

func testHonorGenerate(ctrl *controller.HonorController, cmd string) error {
	raw := strings.TrimSpace(cmd)
	var query model.HonorQuery
	if err := loadQueryFromFile(raw, &query); err != nil {
		return err
	}
	req, err := ctrl.BuildHonorRequest(query)
	if err != nil {
		return fmt.Errorf("build honor request failed: %w", err)
	}
	start := time.Now()
	imageData, err := ctrl.RenderHonorImage(req)
	if err != nil {
		return fmt.Errorf("render honor failed: %w", err)
	}
	fmt.Printf("Render honor success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("honor", 0, imageData)
}

func testProfileGenerate(ctrl *controller.ProfileController, userData *service.UserDataService, cmd string) error {
	userID := strings.TrimSpace(cmd)
	if userID == "" {
		userID = "1"
	}
	region := "jp" // 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻娑樷槈濮楀牊鏁鹃梺鍛婄懃缁绘﹢寮婚敐澶婄闁挎繂妫Λ鍕⒑閸濆嫷鍎庣紒鑸靛哺瀵鈽夊Ο閿嬵潔濠殿喗顨呴悧濠囧极妤ｅ啯鈷戦柛娑橈功閹冲啰绱掔紒姗堣€跨€殿喖顭烽弫鎰緞婵犲嫷鍚呴梻浣瑰缁诲倸螞椤撶倣娑㈠礋椤栨稈鎷洪梺鍛婄箓鐎氱兘宕曟惔锝囩＜闁兼悂娼ч崫铏光偓娈垮枦椤曆囧煡婢跺á鐔兼煥鐎ｅ灚缍屽┑鐘愁問閸犳銆冮崨瀛樺亱濠电姴娲ら弸浣肝旈敐鍛殲闁抽攱鍨块弻娑樷槈濮楀牆濮涢梺鐟板暱閸熸壆妲愰幒鏃傜＜婵鐗愰埀顒冩硶閳ь剚顔栭崰鏍€﹂悜钘夋瀬闁归偊鍘肩欢鐐测攽閻樻彃顒㈢紓宥呭€垮缁樻媴閾忕懓绗￠梺鎸庣娣囧﹪顢涘鎯т紣闁捐崵鍋熼埀顒€鍘滈崑鎾绘煕閺囥劌鍘撮柟閿嬫そ閺屸剝寰勭仦鎴掓勃婵犮垻鎳撳Λ婵婃闂佹寧娲栭崐褰掓偂閵夆晜鐓涢柛銉ｅ劚婵℃椽姊洪褍鐏ｉ柍褜鍓氶鏍窗閺嶎厸鈧箓鎮滈挊澶婄€俊銈忕到閸燁偆绮诲☉妯忓綊鏁愰崨顔兼殘闂佺顭崹璺侯潖閾忚鍏滈柛娑卞幒濮规鏌ｉ悙瀵糕棨闁稿海鏁诲畷娲焵椤掍降浜滈柟鍝勬娴滃墽绱掗崜褑妾搁柛娆忓暣閵嗕礁鈻庨幘宕囩杸濡炪倖鏌ㄦ晶浠嬫晬濠靛鈷戠紒瀣濠€浼存煟閻旀繂娲ょ粈澶屸偓骞垮劚椤︿即鎮￠弴銏＄厸闁告劧绲芥禍楣冩⒑缂佹﹩娈樺┑鐐╁亾婵犵鍓濋幐鍐茬暦濮椻偓椤㈡瑩鎮℃惔銈傚亾閸喒鏀介柍钘夋閻忕娀鏌ｉ妸褍鍘寸€殿喓鍔戦幖褰掝敃閵堝洨妲囬梻浣圭湽閸ㄨ棄顭囪閻☆厽绻濋悽闈涗粶闁活亙鍗冲畷鎰板即閻忕粯妞藉畷绋课旀担鍙夊濠电偠鎻徊浠嬪箟閿熺姴鐤柣鎰湴閳ь剚甯掗～婵嬵敇瑜庨悿渚€鎮楃憴鍕８闁搞劋绮欓獮鎰節閸愩劎绐為梺绯曞墲椤洦绂嶉弽顓熲拻濞达絽鎲￠幆鍫ユ偠濮樼厧浜扮€规洘娲熷濠氬Ψ閵壯冨箞闂備胶顢婇幓顏堟⒔閸曨垱鍋傞柡鍥╁枔缁♀偓闂傚倸鐗婄粙鎺椝夊鍕╀簻闊浄绲肩花濂告煏閸パ冾伃濠碉紕鏌夐ˇ鎻捗归悩娆忔噽绾惧ジ寮堕崼娑樺婵炴惌鍠楅妵鍕閿涘嫭鍣伴梺纭呮珪閻楃娀宕洪悙渚晢濠电姴瀚崑褏绱撴担浠嬪摵閻㈩垱甯￠幃鎯р攽鐎ｎ亞顦ㄥ銈嗘煥閸氬骞婂▎鎴犵＝闁稿本鐟ㄩ崗宀€绱掗鍛仯闁瑰箍鍨藉畷鐓庘攽閸喐顔曢梻浣稿閸嬪懎煤瀹ュ鐒垫い鎺嶈兌婢у灚銇勯姀锛勫⒌妞ゃ垺顨婇崺鈧い鎺嗗亾閻撱倝鏌ㄩ弴鐐测偓褰掑磻閸岀偞鐓涢柛銉㈡櫅閺嬫梻绱掗悩鑽ょ暫闁哄瞼鍠撻埀顒佺⊕椤洨绮婚幘缁樼厾婵繂鐭堝鎰亜閵婏絽鍔﹂柟顔界懇楠炴牠顢橀悢鐑樻缂傚倸鍊烽懗鍓佸垝椤栫偞鍎庢い鏍ㄦ皑閺嗭箓鏌ｉ幘宕囧哺闁衡偓娴犲鐓曟い鎰剁悼缁犳﹢鏌涘Ο鍨撶紒缁樼箓閳绘捇宕归鐣屼邯闂備焦瀵уú蹇涘磹濠靛宓侀柡宥庡弾閺佸啴鏌曢崼婵囧櫣闁汇倝绠栧缁樻媴閽樺鎯炴繝娈垮枟濞兼瑧鍙呴梺鎸庢磻闂勫秵绂嶈ぐ鎺撶厱闁规澘鍚€缁ㄨ棄鈽夐幘宕囆ｇ紒缁樼☉椤斿繘顢欓懡銈呭毈闂備焦濞婇弨閬嶅垂閸ф钃熼柕鍫濇閸欏繘鏌熼柇锕€鍘撮柛瀣崌瀵挳濮€閻樺疇绶㈤梻鍌欑贰閸撴瑧绮旈悽鍛婂亗婵炴垯鍨洪悡鏇熺箾閹存繂鑸归柣蹇曞Х閳ь剚绋掔换鍌炩€旈崘顔嘉ч柛鈩冾焽閳规稓绱撴担鐟扮祷濠⒀傜矙楠炴垿濮€閻橆偅鏂€闂佺硶妾ч弲婊堝磽闂堟侗娓婚柕鍫濇鐏忕敻鏌涚€ｎ偒鐒介柤楦块哺缁绘繂顫濋娑欏濠电偠鎻徊鑲╂媰閿曞倹鍊块柛鎾楀懐锛滅紓鍌欑劍閿氭繛鎼枤閳ь剝顫夊ú蹇涘垂婵傛潌鍥亹閹烘挾鍘遍梺瀹狀潐閸庤櫕绂嶉悙顑跨箚闁绘劦浜滈埀顒佺墱閺侇噣骞掑Δ鈧壕褰掓煠婵劕鈧牠寮冲鍕闁瑰鍋戦埀顒佸笚閹棃濡搁妷褍鈧偤鎮峰鍕疄闁绘搩鍓熼、妤佹媴閸忓摜鐩庨梻渚€娼ф蹇曟閺囩伝娲箻椤旂晫鍘靛銈嗘⒒閺咁偊骞婇崨瀛樼厓鐟滄粓宕滃杈╃煓闁圭儤銇涢埀顒婄畵瀹曞爼顢旈崒娆愮潖闂備礁鎲＄缓鍧楀磿瀹曞洤顥氬ù鐘差儐閻撴洟鎮橀悙鎻掆挃闁瑰啿鎳橀幃褰掑箛椤忓嫬绁梺鍝勬湰閻╊垶骞冮埡浣烘殾闁搞儜鈧幏浼存⒒娴ｇ瓔鍤冮柛鐘虫崌瀹曟洟鎮界粙璺ㄧ暫闂佹枼鏅涢崯顖炲磿閻旀悶浜滈柡鍐ㄥ€瑰▍鏇犵磼鐎ｂ晝鍔嶇紒缁樼箞閹粙妫冨☉妤冩崟婵犵妲呴崹浼存晝閵壯呯焿鐎广儱鎳夐弨浠嬫倵閿濆骸浜滈柍褜鍓欓悘婵嬪煘閹达附鍋愮€规洖娴傞弳锟犳⒑缂佹ɑ灏甸柛鐘崇墵瀵鎮㈤崗鑲╁姺闂佹寧娲嶉崑鎾绘煟韫囨洖鈻堥柡宀嬬畵瀹曠喖顢曢悩韫垝闂備浇顕栭崰妤呫€冮崨杈剧稏婵犻潧顑愰弫鍥煟閺傛寧鎯堥柛銈嗗笚娣囧﹪鎮欓鍕ㄥ亾閵堝鐭楅柍褜鍓熼弻娑欐償閳藉棛鍚嬮柦妯荤箞閺岀喓绱掑Ο杞板垔闂佺粯鏌ㄥΛ娑㈠箟濮濆瞼鐤€婵炴垶顭囬弻褍顪冮妶鍡楀闁圭绻濋弫鎰緞婵犲嫮鏉搁梻浣告惈椤﹀啿鈻旈弴銏╂晩濠电姴鍊甸弨浠嬪箳閹惰棄纾归柟鎯у閻棗霉閿濆懏璐℃い鈺勫皺閹叉悂鎮ч崼婵堢懖闂佹娊鏀遍崹鍧楀蓟閻旂⒈鏁嶉柛鈩冾殔椤忣偅銇勯埡鈧崡鍐差潖閾忓湱鐭欐繛鍡樺劤閸撲即鏌ｆ惔銏犲毈闁革綇绲介悾鐑芥偋閸垻鐦堥梺绋胯閸婃洟骞冮幋鐐电瘈闁靛骏绲剧涵鐐亜閹存繂鈧潡鐛箛娑欏€绘俊顖濆亹椤旀洟姊洪崫鍕垫Ч闁搞劎鏁婚獮鍐箣閿旂晫鍘介梺缁樻⒐缁诲倿骞婇幇鐗堝剹闁圭偓鐣禍婊堟煛閸ヮ煈娈斿ù婊堢畺濮婃椽宕崟顓犲姽缂備胶绮换鍫濈暦濞差亜鐒垫い鎺嶉檷娴滄粓鏌熼崫鍕ら柛鏂跨Ф缁辨挸顓奸崨顔兼灎闂佸搫鑻粔鐟扮暦椤愶箑绀嬫い鎰剁稻椤斿秴鈹戦悩顔肩伇婵炲绋撻埀顒佺濠㈡﹢锝炶箛娑欏€绘俊顖欒閸ゃ倕鈹戦悙鍙夘棡闁搞劎鏁诲畷鍝勭暆閸曨兘鎷哄┑鐐跺蔼椤曆勬櫠椤曗偓閺屾洟宕奸悢绋库偓鎰殽閻愬弶顥犲ù鐙呭閹叉挳鏁愰崱娆屽亾婵犳碍鈷戦柛锔诲幖娴滅偓绻涢崗鑲╂噭缂佸倹甯￠、娑㈡倷缁瀚奸梻浣侯攰閸嬫劙宕戝☉銏犵闁汇垻鏁哥壕濂告煟濡搫鏆遍柣蹇婃櫊閺屽秷顧侀柛鎾村哺閹虫繃銈ｉ崘鈺冨帎闂佹寧绻傞ˇ顖炴嫅閻斿摜绠鹃柟瀛樼懃閻忊晜淇婇锝忚€块柟顔肩秺楠炰線骞掗幋婵愭闁荤喐绮岀粔褰掑春閻愬搫绠ｉ柨鏃囧Г濞呮牠姊洪崨濠冨闁告挻鑹捐闁割偁鍎查崐鐢告偡濞嗗繐顏紒鈧崼銉︾厱闁绘ê纾晶鐢告煏閸℃洜顦﹂柍钘夘槸椤粓宕卞Ο鍝勫帪闂備礁鎼ˇ顖炴偋閸愵喖鐤炬繝濠傛噽缁€濠傘€掑锝呬壕濠殿喖锕ら…宄扮暦閹烘垟鏋庨柟瀛樼箓椤姊绘担绛嬪殐闁哥姵鐗犻幃銉╂偂鎼达絾娈鹃梺鍝勬川娴兼繂煤椤忓秵鏅滈梺鍛婁緱閸樻悂宕戦幘缁樺殝闁汇垻鏁搁惁鍫ユ⒒閸屾氨澧涘〒姘殜瀹曟洝绠涢弴鐘碉紲闂佺粯锚濡﹪鎮℃總鍛婄厵妞ゆ梻鐡斿▓婊勪繆椤愩垹鏆欓柍钘夘槸閳诲酣骞囬浣衡偓鎯р攽?

	start := time.Now()
	imageData, err := ctrl.RenderProfile(userID, region, userData)
	if err != nil {
		return fmt.Errorf("render profile failed: %w", err)
	}
	fmt.Printf("Render profile success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("profile", 0, imageData)
}

func testSKLine(ctrl *controller.SkController, cmd string) error {
	var req map[string]interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderLine(req)
	if err != nil {
		return fmt.Errorf("render sk-line failed: %w", err)
	}
	fmt.Printf("Render sk-line success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("sk_line", 0, imageData)
}

func testStampList(ctrl *controller.StampController, cmd string) error {
	var query model.StampListQuery
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &query); err != nil {
		return err
	}
	req, err := ctrl.BuildStampListRequest(query)
	if err != nil {
		return fmt.Errorf("build stamp-list request failed: %w", err)
	}
	start := time.Now()
	imageData, err := ctrl.RenderStampList(req)
	if err != nil {
		return fmt.Errorf("render stamp-list failed: %w", err)
	}
	fmt.Printf("Render stamp-list success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("stamp_list", 0, imageData)
}

func testMiscCharaBirthday(ctrl *controller.MiscController, cmd string) error {
	var query model.CharaBirthdayRequest
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &query); err != nil {
		return err
	}
	req, err := ctrl.BuildCharaBirthdayRequest(query)
	if err != nil {
		return fmt.Errorf("build misc-chara-birthday request failed: %w", err)
	}
	start := time.Now()
	imageData, err := ctrl.RenderCharaBirthday(req)
	if err != nil {
		return fmt.Errorf("render misc-chara-birthday failed: %w", err)
	}
	fmt.Printf("Render misc-chara-birthday success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("misc_chara_birthday", 0, imageData)
}

func testScoreControl(ctrl *controller.ScoreController, cmd string) error {
	var query model.ScoreControlRequest
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &query); err != nil {
		return err
	}
	req, err := ctrl.BuildScoreControlRequest(query)
	if err != nil {
		return fmt.Errorf("build score-control request failed: %w", err)
	}
	start := time.Now()
	imageData, err := ctrl.RenderScoreControl(req)
	if err != nil {
		return fmt.Errorf("render score-control failed: %w", err)
	}
	fmt.Printf("Render score-control success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("score_control", 0, imageData)
}

func testScoreCustomRoom(ctrl *controller.ScoreController, cmd string) error {
	var query model.CustomRoomScoreRequest
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &query); err != nil {
		return err
	}
	req, err := ctrl.BuildCustomRoomScoreRequest(query)
	if err != nil {
		return fmt.Errorf("build score-custom-room request failed: %w", err)
	}
	start := time.Now()
	imageData, err := ctrl.RenderCustomRoomScore(req)
	if err != nil {
		return fmt.Errorf("render score-custom-room failed: %w", err)
	}
	fmt.Printf("Render score-custom-room success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("score_custom_room", 0, imageData)
}

func testScoreMusicMeta(ctrl *controller.ScoreController, cmd string) error {
	var query []model.MusicMetaRequest
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &query); err != nil {
		return err
	}
	req, err := ctrl.BuildMusicMetaRequest(query)
	if err != nil {
		return fmt.Errorf("build score-music-meta request failed: %w", err)
	}
	start := time.Now()
	imageData, err := ctrl.RenderMusicMeta(req)
	if err != nil {
		return fmt.Errorf("render score-music-meta failed: %w", err)
	}
	fmt.Printf("Render score-music-meta success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("score_music_meta", 0, imageData)
}

func testScoreMusicBoard(ctrl *controller.ScoreController, cmd string) error {
	var query model.MusicBoardRequest
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &query); err != nil {
		return err
	}
	req, err := ctrl.BuildMusicBoardRequest(query)
	if err != nil {
		return fmt.Errorf("build score-music-board request failed: %w", err)
	}
	start := time.Now()
	imageData, err := ctrl.RenderMusicBoard(req)
	if err != nil {
		return fmt.Errorf("render score-music-board failed: %w", err)
	}
	fmt.Printf("Render score-music-board success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("score_music_board", 0, imageData)
}

func testDeckRecommend(ctrl *controller.DeckController, cmd string) error {
	var req map[string]interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderDeckRecommend(req)
	if err != nil {
		return fmt.Errorf("render deck-recommend failed: %w", err)
	}
	fmt.Printf("Render deck-recommend success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("deck_recommend", 0, imageData)
}

func testDeckRecommendAuto(ctrl *controller.DeckController, cmd string) error {
	var query model.DeckAutoQuery
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &query); err != nil {
		return err
	}
	query.RecommendType = normalizeDeckAutoType(query.RecommendType)
	if strings.TrimSpace(query.RecommendType) == "" {
		var fallback map[string]interface{}
		if err := loadQueryFromFile(strings.TrimSpace(cmd), &fallback); err == nil {
			if region, ok := fallback["region"].(string); ok && strings.TrimSpace(query.Region) == "" {
				query.Region = strings.TrimSpace(region)
			}
			if rt, ok := fallback["recommend_type"].(string); ok {
				query.RecommendType = normalizeDeckAutoType(rt)
			}
			if ev, ok := fallback["event_id"].(float64); ok {
				id := int(ev)
				if id > 0 {
					query.EventID = &id
				}
			}
		}
	}
	if strings.TrimSpace(query.RecommendType) == "" {
		query.RecommendType = "event"
	}
	start := time.Now()
	imageData, err := ctrl.RenderDeckRecommendAuto(query)
	if err != nil {
		return fmt.Errorf("render deck-recommend-auto failed: %w", err)
	}
	fmt.Printf("Render deck-recommend-auto success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("deck_recommend_auto", 0, imageData)
}

func testSKQuery(ctrl *controller.SkController, cmd string) error {
	var req map[string]interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderQuery(req)
	if err != nil {
		return fmt.Errorf("render sk-query failed: %w", err)
	}
	fmt.Printf("Render sk-query success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("sk_query", 0, imageData)
}

func testSKCheckRoom(ctrl *controller.SkController, cmd string) error {
	var req map[string]interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderCheckRoom(req)
	if err != nil {
		return fmt.Errorf("render sk-check-room failed: %w", err)
	}
	fmt.Printf("Render sk-check-room success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("sk_check_room", 0, imageData)
}

func testSKSpeed(ctrl *controller.SkController, cmd string) error {
	var req map[string]interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderSpeed(req)
	if err != nil {
		return fmt.Errorf("render sk-speed failed: %w", err)
	}
	fmt.Printf("Render sk-speed success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("sk_speed", 0, imageData)
}

func testSKPlayerTrace(ctrl *controller.SkController, cmd string) error {
	var req map[string]interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderPlayerTrace(req)
	if err != nil {
		return fmt.Errorf("render sk-player-trace failed: %w", err)
	}
	fmt.Printf("Render sk-player-trace success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("sk_player_trace", 0, imageData)
}

func testSKRankTrace(ctrl *controller.SkController, cmd string) error {
	var req map[string]interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderRankTrace(req)
	if err != nil {
		return fmt.Errorf("render sk-rank-trace failed: %w", err)
	}
	fmt.Printf("Render sk-rank-trace success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("sk_rank_trace", 0, imageData)
}

func testSKWinrate(ctrl *controller.SkController, cmd string) error {
	var req map[string]interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderWinrate(req)
	if err != nil {
		return fmt.Errorf("render sk-winrate failed: %w", err)
	}
	fmt.Printf("Render sk-winrate success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("sk_winrate", 0, imageData)
}

func testMysekaiResource(ctrl *controller.MysekaiController, cmd string) error {
	var req interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderResource(req)
	if err != nil {
		return fmt.Errorf("render mysekai-resource failed: %w", err)
	}
	fmt.Printf("Render mysekai-resource success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("mysekai_resource", 0, imageData)
}

func testMysekaiFixtureList(ctrl *controller.MysekaiController, cmd string) error {
	var req interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderFixtureList(req)
	if err != nil {
		return fmt.Errorf("render mysekai-fixture-list failed: %w", err)
	}
	fmt.Printf("Render mysekai-fixture-list success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("mysekai_fixture_list", 0, imageData)
}

func testMysekaiFixtureDetail(ctrl *controller.MysekaiController, cmd string) error {
	var req interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderFixtureDetail(req)
	if err != nil {
		return fmt.Errorf("render mysekai-fixture-detail failed: %w", err)
	}
	fmt.Printf("Render mysekai-fixture-detail success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("mysekai_fixture_detail", 0, imageData)
}

func testMysekaiDoorUpgrade(ctrl *controller.MysekaiController, cmd string) error {
	var req interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderDoorUpgrade(req)
	if err != nil {
		return fmt.Errorf("render mysekai-door-upgrade failed: %w", err)
	}
	fmt.Printf("Render mysekai-door-upgrade success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("mysekai_door_upgrade", 0, imageData)
}

func testMysekaiMusicRecord(ctrl *controller.MysekaiController, cmd string) error {
	var req interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderMusicRecord(req)
	if err != nil {
		return fmt.Errorf("render mysekai-music-record failed: %w", err)
	}
	fmt.Printf("Render mysekai-music-record success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("mysekai_music_record", 0, imageData)
}

func testMysekaiTalkList(ctrl *controller.MysekaiController, cmd string) error {
	var req interface{}
	if err := loadQueryFromFile(strings.TrimSpace(cmd), &req); err != nil {
		return err
	}
	start := time.Now()
	imageData, err := ctrl.RenderTalkList(req)
	if err != nil {
		return fmt.Errorf("render mysekai-talk-list failed: %w", err)
	}
	fmt.Printf("Render mysekai-talk-list success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("mysekai_talk_list", 0, imageData)
}

func testCardListHardcoded(ctrl *controller.CardController) error {
	ids := []int{190, 1252, 1309, 17}
	region := "jp"
	start := time.Now()
	imageData, err := ctrl.RenderCardListFromIDs(ids, region)
	if err != nil {
		return fmt.Errorf("render list failed: %w", err)
	}
	fmt.Printf("Render list success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("card_list_hardcoded", 0, imageData)
}

func testCardListDynamic(ctrl *controller.CardController, cmd string) error {
	raw := preprocessCommand(cmd, "/查卡", "查卡", "查牌", "查卡片", "查询卡片")
	raw = stripLeadingRegionToken(raw)
	queries := []model.CardQuery{{Query: raw, UserID: "test_user"}}

	start := time.Now()
	imageData, err := ctrl.RenderCardList(queries)
	if err != nil {
		return fmt.Errorf("render list failed: %w", err)
	}
	fmt.Printf("Render list success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("card_list_search", 0, imageData)
}

func testCardBox(ctrl *controller.CardController, cmd string) error {
	raw := preprocessCommand(cmd, "/查卡", "查卡", "查牌", "查卡片", "查询卡片")
	raw = stripLeadingRegionToken(raw)
	queries := []model.CardQuery{{Query: raw, UserID: "test_user"}}

	start := time.Now()
	imageData, err := ctrl.RenderCardBox(queries)
	if err != nil {
		return fmt.Errorf("render card box failed: %w", err)
	}
	fmt.Printf("Render card box success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("card_box", 0, imageData)
}

func testCardDetail(ctrl *controller.CardController, parser *service.CardParser, cmd string) error {
	raw := preprocessCommand(cmd, "/查卡", "查卡", "查牌", "查卡片", "查询卡片")
	raw = stripLeadingRegionToken(raw)
	fmt.Printf("Processing command: '%s'\n", raw)

	if _, err := parser.Parse(raw); err != nil {
		return fmt.Errorf("parser failed: %w", err)
	}

	start := time.Now()
	imageData, err := ctrl.RenderCardDetail(model.CardQuery{Query: raw, UserID: "test_user"})
	if err != nil {
		return fmt.Errorf("render failed: %w", err)
	}
	fmt.Printf("Render success! Took %v. Image size: %d bytes\n", time.Since(start), len(imageData))
	return saveImage("card_detail", 0, imageData)
}

func saveImage(prefix string, id int, data []byte) error {
	outputDir := globalOutputDir
	if outputDir == "" {
		outputDir = "D:/github/testfile"
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	filename := fmt.Sprintf("%s_%s_%d.png", prefix, time.Now().Format("20060102_150405"), id)
	outputPath := filepath.Join(outputDir, filename)
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		fallback := filepath.Join(os.TempDir(), filename)
		if writeErr := os.WriteFile(fallback, data, 0o644); writeErr != nil {
			return fmt.Errorf("failed to write image: %w (fallback: %v)", err, writeErr)
		}
		fmt.Printf("Image saved to fallback path: %s\n", fallback)
		return nil
	}

	fmt.Printf("Image saved to: %s\n", outputPath)
	return nil
}

func parseMusicProgressCommand(cmd string) (model.MusicProgressQuery, error) {
	raw := strings.TrimSpace(cmd)
	if strings.HasSuffix(strings.ToLower(raw), ".json") {
		var query model.MusicProgressQuery
		if err := loadQueryFromFile(raw, &query); err != nil {
			return model.MusicProgressQuery{}, err
		}
		if query.Difficulty == "" {
			query.Difficulty = "master"
		}
		if query.Region == "" {
			query.Region = "jp"
		}
		return query, nil
	}

	raw = strings.TrimSpace(strings.TrimPrefix(raw, "/"))
	raw = strings.Replace(raw, "谱面进度", "", 1)
	raw = strings.Replace(raw, "查谱面进度", "", 1)
	raw = strings.Replace(raw, "打歌进度", "", 1)
	raw = strings.Replace(raw, "pjsk进度", "", 1)
	region := "jp"
	fields := strings.Fields(raw)
	if len(fields) > 0 {
		if parsedRegion, ok := map[string]bool{"jp": true, "en": true, "cn": true, "tw": true, "kr": true}[strings.ToLower(fields[0])]; ok && parsedRegion {
			region = strings.ToLower(fields[0])
			fields = fields[1:]
		}
	}
	diff := "master"
	if len(fields) > 0 {
		if normalized, ok := normalizeDifficultyAlias(fields[0]); ok {
			diff = normalized
		}
	}

	return model.MusicProgressQuery{
		Difficulty: diff,
		Region:     region,
	}, nil
}

func parseMusicChartCommand(cmd string) (model.MusicChartQuery, error) {
	raw := preprocessCommand(cmd, "/谱面预览", "谱面预览", "查谱面", "查谱", "谱面", "查谱图", "chart")
	region, cleaned := extractLeadingRegionToken(raw)
	if region == "" {
		region = "jp"
	}
	raw = cleaned
	if raw == "" {
		return model.MusicChartQuery{}, fmt.Errorf("please provide music keyword")
	}
	diff := "master"
	skill := false
	style := ""
	var terms []string
	for _, token := range strings.Fields(raw) {
		if normalized, ok := normalizeDifficultyAlias(token); ok {
			diff = normalized
			continue
		}
		lt := strings.ToLower(token)
		switch lt {
		case "skill", "技能", "withskill":
			skill = true
			continue
		}
		if strings.HasPrefix(lt, "style=") {
			style = strings.TrimPrefix(token, "style=")
			continue
		}
		terms = append(terms, token)
	}
	if len(terms) == 0 {
		return model.MusicChartQuery{}, fmt.Errorf("music keyword missing")
	}
	return model.MusicChartQuery{
		Query:      strings.Join(terms, " "),
		Region:     region,
		Difficulty: diff,
		Skill:      skill,
		Style:      style,
	}, nil
}

func parseMusicRewardsDetailCommand(cmd string) (model.MusicRewardsDetailQuery, error) {
	raw := strings.TrimSpace(cmd)
	if !strings.HasSuffix(strings.ToLower(raw), ".json") {
		return model.MusicRewardsDetailQuery{}, fmt.Errorf("please provide a JSON file path for rewards detail data")
	}
	var query model.MusicRewardsDetailQuery
	if err := loadQueryFromFile(raw, &query); err != nil {
		return model.MusicRewardsDetailQuery{}, err
	}
	if query.Region == "" {
		query.Region = "jp"
	}
	query.ComboRewards = ensureDetailComboRewardsMap(query.ComboRewards)
	return query, nil
}

func parseMusicRewardsBasicCommand(cmd string) (model.MusicRewardsBasicQuery, error) {
	raw := strings.TrimSpace(cmd)
	if !strings.HasSuffix(strings.ToLower(raw), ".json") {
		return model.MusicRewardsBasicQuery{}, fmt.Errorf("please provide a JSON file path for rewards basic data")
	}
	var query model.MusicRewardsBasicQuery
	if err := loadQueryFromFile(raw, &query); err != nil {
		return model.MusicRewardsBasicQuery{}, err
	}
	if query.Region == "" {
		query.Region = "jp"
	}
	if query.ComboRewards == nil {
		query.ComboRewards = map[string]string{
			"hard":   "0",
			"expert": "0",
			"master": "0",
			"append": "0",
		}
	}
	return query, nil
}

func parseMusicListCommand(cmd string) (model.MusicListQuery, error) {
	raw := strings.TrimSpace(strings.TrimPrefix(cmd, "/"))
	replacements := []string{
		"pjsk song list", "pjsk music list", "pjsk music constant",
		"乐曲列表", "乐曲一览", "难度排行", "查乐曲",
	}
	lower := strings.ToLower(raw)
	for _, rep := range replacements {
		lower = strings.ReplaceAll(lower, strings.ToLower(rep), "")
	}

	includeLeaks := strings.Contains(lower, "leak")
	if includeLeaks {
		lower = strings.ReplaceAll(lower, "leak", "")
	}

	tokens := strings.Fields(lower)
	region := "jp"
	if len(tokens) > 0 {
		if _, ok := map[string]struct{}{"jp": {}, "en": {}, "cn": {}, "tw": {}, "kr": {}}[tokens[0]]; ok {
			region = tokens[0]
			tokens = tokens[1:]
		}
	}
	diff := "master"
	if len(tokens) > 0 {
		if normalized, ok := normalizeDifficultyAlias(tokens[0]); ok {
			diff = normalized
			tokens = tokens[1:]
		}
	}

	var levels []int
	for _, token := range tokens {
		if n, err := strconv.Atoi(token); err == nil {
			levels = append(levels, n)
		}
	}

	query := model.MusicListQuery{
		Difficulty:   diff,
		Region:       region,
		IncludeLeaks: includeLeaks,
	}

	switch len(levels) {
	case 1:
		query.Level = levels[0]
	case 2:
		query.LevelMin = levels[0]
		query.LevelMax = levels[1]
	}

	return query, nil
}

func normalizeDifficultyAlias(token string) (string, bool) {
	token = strings.TrimSpace(strings.ToLower(token))
	switch token {
	case "easy", "ez":
		return "easy", true
	case "normal", "nm":
		return "normal", true
	case "hard", "hd":
		return "hard", true
	case "expert", "exp", "ex":
		return "expert", true
	case "master", "mas", "ma":
		return "master", true
	case "append", "apd":
		return "append", true
	default:
		return "", false
	}
}

func loadQueryFromFile(path string, target interface{}) error {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return err
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	return json.Unmarshal(data, target)
}

func normalizeDeckAutoType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "event_pt", "event":
		return "event"
	case "bonus", "event_bonus":
		return "bonus"
	case "challenge", "no_event", "mysekai":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return ""
	}
}

func ensureDetailComboRewardsMap(combo map[string][]model.MusicComboReward) map[string][]model.MusicComboReward {
	if combo == nil {
		combo = make(map[string][]model.MusicComboReward)
	}
	for _, diff := range []string{"hard", "expert", "master", "append"} {
		if _, ok := combo[diff]; !ok {
			combo[diff] = []model.MusicComboReward{}
		}
	}
	return combo
}

func parseGachaListCommand(cmd string) model.GachaListQuery {
	query := model.GachaListQuery{
		Region:      "jp",
		Page:        1,
		PageSize:    6,
		IncludePast: true,
	}

	raw := strings.TrimSpace(strings.TrimPrefix(cmd, "/"))
	replacements := []string{
		"pjsk gacha", "卡池列表", "卡池一览", "查卡池", "卡池",
	}
	lower := strings.ToLower(raw)
	for _, rep := range replacements {
		lower = strings.ReplaceAll(lower, strings.ToLower(rep), "")
	}
	if parsedRegion, rest := extractLeadingRegionToken(lower); parsedRegion != "" {
		query.Region = parsedRegion
		lower = rest
	}

	for _, token := range strings.Fields(lower) {
		t := strings.TrimSpace(token)
		if t == "" {
			continue
		}
		lt := strings.ToLower(t)
		switch {
		case strings.HasPrefix(lt, "p"):
			if val, err := strconv.Atoi(strings.TrimPrefix(lt, "p")); err == nil && val > 0 {
				query.Page = val
			}
		case lt == "leak":
			query.IncludeFuture = true
		case lt == "当前" || lt == "current":
			query.OnlyCurrent = true
			query.IncludeFuture = false
			query.IncludePast = false
		case lt == "复刻" || lt == "rerelease" || lt == "back":
			query.IsRerelease = true
		case lt == "回响" || lt == "recall":
			query.IsRecall = true
		case lt == "past":
			query.IncludePast = true
		case lt == "nopast":
			query.IncludePast = false
		case strings.HasPrefix(lt, "card"):
			if val, err := strconv.Atoi(lt[4:]); err == nil {
				query.CardID = val
			}
		default:
			if val, err := strconv.Atoi(lt); err == nil {
				if val >= 2000 && val <= 2100 {
					query.Year = val
				} else {
					query.CardID = val
				}
			} else {
				query.Keyword = strings.TrimSpace(t)
			}
		}
	}
	return query
}

func parseGachaDetailCommand(cmd string) (model.GachaDetailQuery, error) {
	raw := strings.TrimSpace(strings.TrimPrefix(cmd, "/"))
	replacements := []string{
		"卡池",
		"查卡池",
		"抽卡",
		"查看卡池",
		"gacha",
		"gacha-detail",
		"gachadetail",
		"pool",
		"banner",
	}
	for _, rep := range replacements {
		if rep == "" {
			continue
		}
		raw = strings.ReplaceAll(raw, rep, "")
	}
	raw = strings.TrimSpace(raw)
	region, cleaned := extractLeadingRegionToken(raw)
	if region == "" {
		region = "jp"
	}
	raw = cleaned
	if strings.HasPrefix(raw, "-") {
		if idx, err := strconv.Atoi(raw); err == nil && idx < 0 {
			return model.GachaDetailQuery{
				Region:   region,
				GachaID:  -1,
				NegIndex: -idx,
			}, nil
		}
	}
	if strings.HasPrefix(raw, "event") {
		if eid, err := strconv.Atoi(raw[5:]); err == nil {
			return model.GachaDetailQuery{
				Region:  region,
				GachaID: -2,
				EventID: eid,
			}, nil
		}
	}
	id, err := strconv.Atoi(raw)
	if err != nil {
		return model.GachaDetailQuery{}, fmt.Errorf("invalid gacha id: %s", raw)
	}
	return model.GachaDetailQuery{
		Region:  region,
		GachaID: id,
	}, nil
}

func parseEventListCommand(cmd string, parser *service.EventParser) (model.EventListQuery, error) {
	query := model.EventListQuery{
		Region:        "jp",
		IncludePast:   true,
		IncludeFuture: true,
		Limit:         6,
	}
	raw := preprocessEventCommand(cmd)
	if parsedRegion, rest := extractLeadingRegionToken(raw); parsedRegion != "" {
		query.Region = parsedRegion
		raw = rest
	}
	if raw == "" {
		return query, nil
	}
	tokens := strings.Fields(raw)
	var filtered []string
	for _, token := range tokens {
		lower := strings.ToLower(token)
		switch {
		case lower == "list" || lower == "列表" || lower == "一览":
			continue
		case lower == "past":
			query.IncludePast = true
		case lower == "future":
			query.IncludeFuture = true
		case lower == "onlyfuture":
			query.OnlyFuture = true
			query.IncludeFuture = true
			query.IncludePast = false
		case strings.HasPrefix(lower, "limit"):
			if v, err := strconv.Atoi(strings.TrimPrefix(lower, "limit")); err == nil && v > 0 {
				query.Limit = v
			}
		default:
			filtered = append(filtered, token)
		}
	}
	filteredRaw := strings.TrimSpace(strings.Join(filtered, " "))
	if filteredRaw == "" {
		return query, nil
	}
	info, err := parser.Parse(filteredRaw)
	if err != nil {
		return query, err
	}
	if info.Type != service.QueryTypeEventFilter {
		return query, fmt.Errorf("please provide filters like 'wl' or '25h 2024'")
	}
	query.EventType = info.Filter.EventType
	query.Unit = info.Filter.Unit
	query.Attr = info.Filter.Attr
	query.Year = info.Filter.Year
	query.CharacterID = info.Filter.CharacterID
	return query, nil
}

func preprocessEventCommand(cmd string) string {
	cmd = strings.TrimSpace(strings.TrimPrefix(cmd, "/"))
	replacements := []string{
		"event-list",
		"event",
		"events",
		"pjsk event",
		"pjsk events",
		"活动",
		"查活动",
		"活動",
		"查活動",
		"活动列表",
		"活动详情",
		"查活动列表",
	}
	lower := strings.ToLower(cmd)
	for _, rep := range replacements {
		if rep == "" {
			continue
		}
		lower = strings.ReplaceAll(lower, strings.ToLower(rep), "")
	}
	return strings.TrimSpace(lower)
}

type educationInputKind int

const (
	educationInputAuto educationInputKind = iota
	educationInputJSON
	educationInputRegion
)

func determineEducationInputType(cmd string) educationInputKind {
	raw := strings.TrimSpace(cmd)
	if raw == "" {
		return educationInputAuto
	}
	lower := strings.ToLower(raw)
	if strings.HasSuffix(lower, ".json") {
		return educationInputJSON
	}
	normalized := parseEducationRegion(raw)
	if normalized == "" {
		return educationInputAuto
	}
	return educationInputRegion
}

func parseEducationRegion(cmd string) string {
	raw := preprocessCommand(cmd,
		"/education-challenge", "education-challenge",
		"/education", "education",
		"/challenge", "challenge",
		"/挑战信息", "挑战信息",
		"/挑战", "挑战",
		"/教育挑战", "教育挑战",
	)
	return strings.ToLower(strings.TrimSpace(raw))
}
