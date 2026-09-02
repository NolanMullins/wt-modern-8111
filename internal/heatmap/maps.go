package heatmap

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math/bits"
	"strconv"

	"github.com/disintegration/imaging"
)

const mapHashTolerance = 42

type Map struct {
	Level string
	Name  string
}

type mapHash struct {
	hash string
	id   string
	name string
}

// Ground-map hashes are adapted from ElBartt/WarThunder-Plotter under its MIT
// license. See THIRD_PARTY_NOTICES.txt.
var groundMapHashes = []mapHash{
	{"2a42552469888e3091a113c4230a861b4e7958312a489687084430690", "avg_abandoned_factory", "Abandoned Factory"},
	{"932c3b5b5aa14cf5c7ff66dc47a62f3d2f2c683c55dd45644758da288", "avg_abandoned_town", "Abandoned Town"},
	{"d265b19ba67427394ebe125f5567086f16a71959c26958e999d4e59a0", "avg_africa_desert", "El Alamein"},
	{"acae989d893a9a7174e2b6457680ed15de1bfc5579cb730b6e27c94f8", "avg_alaska_town", "Alaska"},
	{"b71e46f98d9133946f2c6c78e861b9c665830f0d3a3536776696e72b0", "avg_american_valley", "American Desert"},
	{"716ac8c5b750ad5adc34728b551e13bc8ef11cd9b9ab73c56ed96d938", "avg_aral_sea", "Aral Sea"},
	{"4b18aa6115e64dca8f945b4625523630fee0e98bda0774a6cd5489ac8", "avg_arctic", "Arctic"},
	{"383ab9ef565ae3331caeb7185924b9a6b054e35586951d2edaacace98", "avg_ardennes_snow", "Ardennes (winter)"},
	{"381ed16f565ae53b1eaeb758792cb8a69154a35596911d2e5afcace98", "avg_ardennes", "Ardennes"},
	{"914cb995cb169c8845b34a84cc2192c5846b98a54dc6585ab87559230", "avg_berlin", "Berlin"},
	{"f169c55310971598abc45bc2a5c16b48ad4caf4e5f51db615386d99c8", "avg_breslau", "Breslau"},
	{"d909fa6696cc1e9127a8e740e601ec83b809721af017d43544af8c458", "avg_container_port", "Cargo Port"},
	{"08c01242118066032c8cf009e235807241631c341ad89631b49e25380", "avg_eastern_europe", "Eastern Europe"},
	{"b3628d89b37141d26df493ef75cd4b01975beeb5dd8bc436ca234a438", "avg_egypt_sinai", "Sinai"},
	{"6ca0ed414278e4fdd1f8b3e2b7e66ac64d911f96a925e64691a5070a8", "avg_european_fortress", "White Rock Fortress"},
	{"3731b8f6136c487805e097c527805f049801a0053009082490f021910", "avg_finland", "Finland"},
	{"904f04e81a9023b102e40f883e107c01f007591f325ca3f887f11f860", "avg_fulda", "Fulda"},
	{"48ce196cb0caad90c72436d2ad85455a9b673637e4e19a736994c6318", "avg_greece", "Greece"},
	{"f97c7bbcea78ea321a26a742399d2b1156ab3184d0c999b9f155d0590", "avg_guadalcanal", "Guadalcanal"},
	{"487223c5e5b30662794961cb6f1e9c7f38f720f723e9cf32198424e00", "avg_hurtgen", "Hurtgen Forest"},
	{"1e2ec457487801b00362035c948523071e06a4051800f00e000000000", "avg_iberian_castle", "Iberian Castle"},
	{"f39f326f6c7ef0f7f236e0ed513395b6a1234f452c6416cdeccb9b3c8", "avg_ireland", "Ireland"},
	{"6e9c9d71bee3dd837e0b7c4edc1db8ba7938f433b97f73d6c6ee8acd8", "avg_israel", "Sun City"},
	{"4f271f013618e7158c831da629d93a31caf895c83b0928b711a6591a0", "avg_japan", "Japan"},
	{"73446f597859e32b8cdb127555666ad07741aadab566c685a168f8d80", "avg_karantan", "Karantan"},
	{"24cd28bf0e5459b99a7394f4af6bfcd2bc3d3a3ddeb37c4f3cf189f98", "avg_karelia_forest_a", "Karelia"},
	{"817f02df07bccb951d0ab812cb83b68f3b9c97597ff0be2c58d9a28d8", "avg_karpaty_passage", "Carpathians"},
	{"882334407a02a2005620a018c4d929da93c987eb077a0fc81d78164c0", "avg_korea_lake", "Korea"},
	{"c7838f873f1c7231c4b3cb6f20fe13f867fb4ff68ff21f267e04f01b8", "avg_krymsk", "Kuban"},
	{"664ccc91ada359263b247074f1f8b9f0f3e0efecf56f48df91fa27ec0", "avg_kursk_villages", "Fire Arc"},
	{"d804902c93344e719ded21d07300f601dea3dc02f807701fa14d55b20", "avg_lazzaro_italy_new_city", "Campania"},
	{"a446af4d64b224acb949518554b9a87bd293342a6c5d2d6c71f042e48", "avg_maginot", "Maginot Line"},
	{"8e5910c071a1f38089c43df0e166c062816208c504e609300c8030000", "avg_mozdok", "Mozdok"},
	{"23f2e789acab6dd56faf9317a636a92d2e5ac93ddb96936af1b3476f8", "avg_netherlands", "Netherlands"},
	{"ed0244be9758cd41d3a59104e293443254a584ee08d610bc40b3d7a68", "avg_normandy", "Normandy"},
	{"3181278dcd6bbcde585999bcb1edd6dae593cb33a6ea0d5cf85570bc0", "avg_northern_india", "Pradesh"},
	{"644e643caa32e4169a0eac32db327169298ad79ac7963b05b94b75568", "avg_northern_valley", "Golden Quarry"},
	{"a4ae989d893a9a7575e2b64576892d165a1bfc5579cb330b6e07c94f8", "avg_nuclear_incident", "Nuclear Incident"},
	{"1c0a80ca5175c2168ea6144c5e1f1c6e341e656cca7f44f083f086fc8", "avg_poland_snow", "Poland (winter)"},
	{"c3f51fc43cc8f9237b18e2b1e3e2c385872c8e113c00f601b69b79150", "avg_poland", "Poland"},
	{"28d313c21e96482d4e117024e05780ae01d8019a13bc818e039801f00", "avg_port_novorossiysk", "Port Novorossiysk"},
	{"718b8a86d6878b1d72929413a42ae597159a42eda7a24c5d2d900c388", "avg_red_desert", "Red Desert"},
	{"9765964f26af6b4eaa5e977e32fc79f86acbc766cd23934eaf836b060", "avg_rheinland", "Advance to the Rhine"},
	{"2410c8309231ae415cca2c942218a6390c623b00d6262c443928f0c88", "avg_sector_montmedy_snow", "Fields of Normandy (winter)"},
	{"2650cc309820a041c4ca29942308a6390862338096261c44f028f0408", "avg_sector_montmedy", "Fields of Normandy"},
	{"9c1d0cc811f45ba87488f213b66d4c365c729cc93989932c00be81dd8", "avg_snow_alps", "Frozen Pass"},
	{"38aa133c12cb6593dc6caddc58b2b32566421de433589645a92589c10", "avg_soviet_range", "Soviet Range"},
	{"c5e125e6c2db25764b8eb72f545c896903c38251a86010d870b0f1500", "avg_soviet_suburban_snow", "Seversk-13 (winter)"},
	{"f2d540c283b992726ad525ce2bb04b61b6572a394b78b339375328750", "avg_soviet_suburban", "Seversk-13"},
	{"238302c18a7ec2b10d627650eda7b137c65f99fee7e78f5c0ef01ec08", "avg_stalingrad_factory", "Stalingrad"},
	{"43d003e603f031f053e257c07f80ff01fe03fc2ff81bf02fc04f903d0", "avg_sweden", "Sweden"},
	{"5d18f638f01869b273add64448098d152eacfa11e19d031a04bc03600", "avg_syria", "Middle East"},
	{"b67f68fca0ea49939636166d0af84df25bc6378c6f89de93b9a1e7650", "avg_training_ground", "Training Ground"},
	{"ce0f3cbe75b7eaffd0f5b9e47f14be8a9d619f633f7af9b3f3c4ef8f8", "avg_tunisia_desert", "Tunisia"},
	{"5b80c5c595d82c7858e30f849b04b68c270c3f0d725ce0cb820f08f78", "avg_vietnam_hills", "Vietnam"},
	{"6672aa6764b258b2dac22f0a23506ab19956dc6da490c964330952548", "avg_vlaanderen", "Flanders"},
	{"2b00cdc8f1627568d60b0a0669b6449c5c8db9c2f0ec645bc383a0488", "avg_volokolamsk", "Volokolamsk"},
	{"296a0eb524aac9d79bc6f605e88eb16d8ad91554a85a1084691d966c8", "avg_western_europe", "Spaceport"},
}

var airInGroundMapHashes = []mapHash{
	{"45265a9f99114921ce3b1d7036242e1ab4166096d29a4859c9b6684d8", "avg_abandoned_factory", "Abandoned Factory"},
	{"222753668118542a95bb07372c74337e627844b5497d426accc8f5f60", "avg_abandoned_town", "Abandoned Town"},
	{"a5921916134ca79a2f2efad96433c2262e9434eaa9e52ad4c5619d270", "avg_alaska_town", "Alaska"},
	{"f017c066a59c43309367754ee2dda44d229f473ea48c6b3c6630cc618", "avg_american_valley", "American Desert"},
	{"99cb316dd6f839397948d76bef2fc73fb24791bf35bcac6aa9a577ea8", "avg_aral_sea", "Aral Sea"},
	{"5896cc19a2344b498d4109923ea36686a09a44d6c146ec45118a04620", "avg_arctic", "Arctic"},
	{"eca3b286229cc0e983b15e48fcb0ed2ef4f527cf6f5ac75d5ad407a98", "avg_ardennes", "Ardennes"},
	{"aca33307221c80ed23f14e49f412cd2c747534cd6f1ac74d5ad405a98", "avg_ardennes_snow", "Ardennes (winter)"},
	{"519282a620a260b86127812d88b5081a2835134e00600aa5950996280", "avg_breslau", "Breslau"},
	{"de6866eb94f1cbe91704ef23cc29bdb3b82ad9b4f4b0e966c78986170", "avg_container_port", "Cargo Port"},
	{"01fc03fe0bd407e20fd48f700f031f071e10294052684e0844029ea58", "avg_eastern_europe", "Eastern Europe"},
	{"0bf52fc73cc330d161c2e3a5ce39886348c611c6264c37987f18bb080", "avg_egypt_sinai", "Sinai"},
	{"e589cb0b479cdb81867689216258cc8e998d3b6a34cf4b83da27ae260", "avg_european_fortress", "White Rock Fortress"},
	{"254821287158c4b827215a631412c23700ec974c52366654c4cc958d0", "avg_finland", "Finland"},
	{"6cc7b183e6678d2e1bda26d4cde8b1c9aab14e25bf0b263a5e7832730", "avg_fulda", "Fulda"},
	{"8e12cb1c60b0a3630ece3d85b30c47300e001c0048c471807200c0008", "avg_greece", "Greece"},
	{"f0edd3f260b07331e382f19b31ec59c49941bf2495812ee0aa9018490", "avg_guadalcanal", "Guadalcanal"},
	{"42bd127c8731a2529435c137224dc7e3958e947118dcbe39292f405b8", "avg_hurtgen", "Hurtgen Forest"},
	{"705d547b56ee907451e2b380a3018e0b1810001100020000000000000", "avg_iberian_castle", "Iberian Castle"},
	{"60e588f229e24b8857041c4afc0df819f813f42b984f738ec65689ad8", "avg_israel", "Sun City"},
	{"11beb07eb545ca71e0e3c446d483948618036104c461a3c30dc61a9c0", "avg_japan", "Japan"},
	{"1c70bae0f1c8e399cf03cf07192e383e70f961fa83f806790cf63d880", "avg_karelia_forest_a", "Karelia"},
	{"66cc4367c677b5c3f91b7b336426e6e253e03190b1909029ae130bb98", "avg_karpaty_passage", "Carpathians"},
	{"4c8a352858b0d0773078d9a19367a065409686cc1d182e308e459c928", "avg_korea_lake", "Korea"},
	{"60e9439887118c631846020408039015205a274020046000402884400", "avg_lazzaro_italy_new_city", "Campania"},
	{"0cb0086418d022a04e08d915100b8217b088481030256043009709220", "avg_maginot", "Maginot Line"},
	{"922596622a659609305571ee871a9275aa22a5984a34386c64c8b1f50", "avg_netherlands", "Netherlands"},
	{"eba21650881bd055e213c2238c9b507548ea80c492c9249c43198e368", "avg_northern_india", "Pradesh"},
	{"737dbde15fc0f80ae035efba91d6469a694b43a72b832b452d9873b08", "avg_northern_valley", "Golden Quarry"},
	{"ed0090707ae1f0c3c79aaf4bc78f1e0d1e9e1c12382c301c200e40148", "avg_poland", "Poland"},
	{"ed0091707ae1f1c3c79aaf4bc78f1e0d1e9e1c12382c301c200e40148", "avg_poland_snow", "Poland (winter)"},
	{"4bba8f6d0fa62b44839b99f18cb3ca278207000c30002000000000000", "avg_port_novorossiysk", "Port Novorossiysk"},
	{"489b331c5a3855d96ab8d331ea29cf56aa0f24162e6c18ea73136a648", "avg_red_desert", "Red Desert"},
	{"b033e4637036e0c9c011e0b3f373c4f58de313e287c70fc65f823f060", "avg_rheinland", "Advance to the Rhine"},
	{"cd891182b125322a3c5d96bd1655033b4c77646e50bad09e981c311b8", "avg_sector_montmedy", "Fields of Normandy"},
	{"cd891082b125322a3c5d96bd165503bb4d77606e509ad09e901d311b8", "avg_sector_montmedy_snow", "Fields of Normandy (winter)"},
	{"6272d5e476469b030d4cd13302e473a1ee86ce4699856d9ba90aea1b0", "avg_snow_alps", "Frozen Pass"},
	{"21206b24331c27928a6cd07b5430a64e2b8568524dce692ac2a1bc668", "avg_soviet_range", "Soviet Range"},
	{"71216243221347048a49a5329c3515c8b16148c55d894d689ea11c440", "avg_soviet_suburban", "Seversk-13"},
	{"33235d44b39b67128a2dc57b9835149a3b456cf15d8c49aaaac17b468", "avg_soviet_suburban_snow", "Seversk-13 (winter)"},
	{"0136026e139c1e7498f15bd097a12f4a3e2e7c01fa6ff9afe317c43f8", "avg_sweden", "Sweden"},
	{"d2c9f123aa2cb0d1638867104e625c79d4cf135a29d4b7b1b2e465920", "avg_syria", "Middle East"},
	{"e1c6837c47d00f807e037c02f08de023e087c08fc10f801f827f80d70", "avg_tunisia_desert", "Tunisia"},
	{"0cbe19fc5f782ef46fe93f9cbf3dfe39fc7270fe46f88c70186034000", "avg_vietnam_hills", "Vietnam"},
	{"4038005401a802c005800bc012802980c3811b826720de600ec029820", "avg_vlaanderen", "Flanders"},
	{"e651c5230296184e989d63ad435a96d52d6658c4d49c4d541a4b38660", "avg_volokolamsk", "Volokolamsk"},
	{"8517855987e6a15916995642d8d313a7313c72596118be39306c95198", "avg_western_europe", "Spaceport"},
}

func ResolveMap(body []byte) (Map, error) {
	hash, err := differenceHash(body)
	if err != nil {
		return Map{}, err
	}
	return resolveHash(hash)
}

func resolveHash(hash string) (Map, error) {
	bestDistance := int(^uint(0) >> 1)
	var best mapHash
	for _, candidate := range append(groundMapHashes, airInGroundMapHashes...) {
		distance, ok := hashDistance(hash, candidate.hash)
		if ok && distance < bestDistance {
			bestDistance = distance
			best = candidate
		}
	}
	if bestDistance > mapHashTolerance {
		return Map{}, fmt.Errorf("%w: closest hash distance %d", ErrUnknownMap, bestDistance)
	}
	return Map{
		Level: "levels/" + best.id + ".bin",
		Name:  best.name,
	}, nil
}

func differenceHash(body []byte) (string, error) {
	source, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("decode map image: %w", err)
	}
	scaled := imaging.Resize(imaging.Grayscale(source), 16, 15, imaging.Lanczos)

	hash := make([]byte, 0, 57)
	nibble := 0
	bitCount := 0
	for y := range 15 {
		for x := range 15 {
			left := color.GrayModel.Convert(scaled.At(x, y)).(color.Gray).Y
			right := color.GrayModel.Convert(scaled.At(x+1, y)).(color.Gray).Y
			nibble = nibble<<1 | boolBit(left > right)
			bitCount++
			if bitCount == 4 {
				hash = strconv.AppendInt(hash, int64(nibble), 16)
				nibble = 0
				bitCount = 0
			}
		}
	}
	if bitCount > 0 {
		nibble <<= 4 - bitCount
		hash = strconv.AppendInt(hash, int64(nibble), 16)
	}
	return string(hash), nil
}

func hashDistance(left, right string) (int, bool) {
	if len(left) != len(right) {
		return 0, false
	}
	distance := 0
	for index := range left {
		a, errA := strconv.ParseUint(left[index:index+1], 16, 4)
		b, errB := strconv.ParseUint(right[index:index+1], 16, 4)
		if errA != nil || errB != nil {
			return 0, false
		}
		distance += bits.OnesCount8(uint8(a ^ b))
	}
	return distance, true
}

func boolBit(value bool) int {
	if value {
		return 1
	}
	return 0
}
