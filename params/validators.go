package params

import (
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

const (
	FirstUpdateValidatorSetHeight  uint64 = 199
	SecondUpdateValidatorSetHeight uint64 = 399
	NetworkSplitStartHeight        uint64 = 398
	NetworkSplitEndHeight          uint64 = 620

	NetworkSplitGroupA      = "A"
	NetworkSplitGroupB      = "B"
	NetworkSplitGroupCommon = "Common"
)

// "0xbcdd0d2cda5f6423e57b6a4dcd75decbe31aecf0": "9f91673ebbc1851f931455c14fe6390c9477d5c0b5a0ab44ff07a5d2f533bb9e", // 0
// "0xbbd1acc20bd8304309d31d8fd235210d0efc049d": "89e90304b7d84d2396a0db628c6311b1051b528a9e2e8e214ce13f9502f1a05d", // 1
// "0x5e2a531a825d8b61bcc305a35a7433e9a8920f0f": "c055fbf4deef68ab511de05154f4415b3d93f96b4ac55d8fd06d6bbf30ca4ea1", // 2
// "0x3ad55d1d552cc55dee90c0faf0335383b2e6c5ce": "fefe0044d84fa6179c329087968e62bb26f04d2b317344de221e379cf4220ecc", // 3
// "0xfe02c8ff2374583c47b1d62fdf3e1b72c20ebe29": "4a5ff76c649ea0ce00cffb65f1509ea76b94a48c73e7060f0908d60ac6222368", // 4
// "0xf7698afa5461438ff438c2322d6d29a5f7abdffd": "73fe8072432096f809ac0719cd93d3dcd70d0a5ba5aaa6940ce9d86df220f251", // 5
// "0x5fda3ff6ea581ea7a5a9c2cb310b13c2126b4e8b": "f291b202160c9e8e56c7bd86d5411076a2268cffefc55d3c30a7fcd97aaa20f6", // 6
// "0xd30d79639bc9c4ed71031bce28216862b80f4b6b": "b8a3e3a3d21fbdc86d6eae456c4c9acde51b426146156f10ec8b1a391b979ffc", // 7
// "0x51cb3d0f6b77ef8317b31f4aaeaa75e4cff3cca7": "c503518590e6507d8ed99c742db712efe77652fc3ee4fa36d0a4cf2237d7dbab", // 8
// "0xabb28e397ae478366271806b4851d81a678e404b": "e4c00c946476729419a49f83c27b916b254dbba155a372bf9c31be27728d32f9", // 9
// "0x6c73f4f3295f83ce342e4a82e8a50d218442451b": "328c3b22adf355eb7346395c71ef3a55dad7a5be1b3f147768554cb4dc506c81", // 10
// "0xc12cf70a667d541a33bd51c623f8a7024ed8c2fe": "e415e62afe6162e0eb02e50e62a25fdd18acc3ba4dbfb15e6279420500dddc3c", // 11
// "0xc12cf70a667d541a33bd51c623f8a7024ed8c2fe": "e415e62afe6162e0eb02e50e62a25fdd18acc3ba4dbfb15e6279420500dddc3c", // 11-b
// "0xd2d3139575c2824d793d1664c2e1aaeecade11c0": "24f58905eb4563bcaa307d8c63da3ebbd862ddeaf4e42fe9383993177529d4c4", // 12
// "0x20be3a44b2ae6be29acf84ed63afe60b09179cdc": "78bf3115c505c7e52373a0724397c043b23d33cfe17577f8b856899bed6526d7", // 13
// "0x50b947c8643c7694037b29545fbc423951e28442": "a967c80f8aac15196308af6676a01a0d09b22ea8b33e6ecff54d4d690ad35cc1", // 14
// "0xa8938f397823afcaa252bb7df137d39396456983": "557de0abfbf8661c8a756ccfcc1537f80d1274baf83fe470c3ab6c913e92fed2", // 15
// "0x5a7ae634876fb264f97eacc24a9261005e9bc39a": "b52c512ef52b76db48665aa3fd75f3edf5da5b3c705706184e39a725718125ad", // 16
// "0x511aa4d222618f8698feaab811023ca4e8bebfe5": "1587e7ff477cb3a8946c41bba115ef405a4f34310dc1ca123e78a9889d88999d", // 17
// "0xe9693a85e563485da999b7d378d60483e89caa0e": "e24af340c044a8df2f3adb3a864bba8621f6ff677da7ce8b6e2c08ce0b3189c5", // 18
// "0x9b50a300da0cd7e036ec2cc12418756ec07004bd": "08302440e4dfc79cd36804be7a90560ce4be0154b9aeb6f61db19f128aa30f5e", // 19
// "0x297e5ebba75bbb67de013eb3d319dd0a2a9861e9": "78430d91e5427a9dabc3288c0d8eb413ddcd349876d18e1c9f445f1a4035185f", // 20
// "0x92e3e1648722a36b975abbfe157297488401770a": "752e87f6ab425d93fec428639ceca87903b61226b19829c2640564e82c6a64fc", // 21
// "0x99303b493062821b6064b79ed3f221f1ea9efba3": "c10bb93b5f9034037dca50e03c1ae83a2d4a3b6f4d62d4ca8a194f0b31ac29da", // 22
// "0xccbb0dac30b4ece3ea897b171992ef07d1e516fc": "2fc71f4bad9b49460d44b5b68324f9a257dad1b5e4b3323e05daecdd3992b94a", // 23
// "0x67ea4bb0a4a72c06861b431faa4700ff21ce1e8a": "7e8cd1146d54ce908fa8c09a446b0b0ae3e3f8e6b0ac18189fe5d3a97540f5e4", // 24
// "0x59b67dc3bfcfd1317cf4a2638ce85cff618cae3f": "ef829e72083bc0ecf0de5fd9adb5ee2eba23488302e2e6de59170e0cf9a7c654", // 25
// "0x1287138fbfcccb306a2440df46405fda6743477e": "8ef39dffe57a42fc9d7cddb1c6288b001a9770d94cd1627d393e1f412de4dfa2", // 26
// "0x48f61af60a5dbd89583d322831d9087c182d7be8": "17344163f156b670de43808b12e67d9c2e985c55d79554c776b65f851be9cc6f", // 27
// "0x8ff775a6efc53740470523d1a4819c7231122de4": "2fe947dd5915bb1b6d0730b8073845d2f32c82635d9f694fdeb8ccbdac53ce83", // 28
// "0xe0e8f47a4b1d600c36bd8b941197c8ed774b6c5e": "44d50313fc7401ca6d6189285afb70e1a8dbafbc7177068729a17be0458a2d22", // 29
// "0x11a1f1b94cc634054f27de1c2aab452db8991da1": "47cfe5d4812985732a6572b0bd346f37087e87439721044f9c9d7d593270c41c", // 30
// "0xcf9de1330157668bf61c5199ebba58b765d859c9": "c3740ef0f90606a242a115b6b710bf4d0deef7d791b61d164409142135a9a97f", // 31
// "0xcb0be46b43456d0f5afa020241da706a0e291faa": "3cc70091c95e888cdea03a4a5b3f5a803ea2dd79ecafa7248990f040c39120ef", // 32
// "0xfd542d7851dc5f7ad7b0f65a85ebaa7560d45201": "8dbe9e590839737d277988605b7b1ff178a15ca92c2014efdd3c9a68f0c6be51", // 33
// "0x4019161102be5b2f610ef20490662308646bda71": "ce67b8ffb4d48a79a1b9c1e699137927248af337c2972e51796a5197e2ca8684", // 34
// "0x057a5760f93e327b92002b59078f309bcc4a08d4": "62d3fb730d7024f2e7a3889b471595b189b2f6a4ca721f652df87ceb1a4bf591", // 35
// "0x1708f58ecfcdab6147ea6ec3e317e880741add97": "e3ae0823a758ee925be0ab4fafbd6ad313a01e13f00ea9b62642bd26182a0239", // 36
// "0x742e985ef0caac0367fe67d486d7d59bdbd895d9": "ee1bdcae747b74027cb44ae1bf80437c413b8cb07a61845ef6d3cc567cd95182", // 37
// "0x46921115b275f7979ba04a9cb7c1c8302924aa3f": "afd6b7bef2301b29f2a0e131340e76ef19c01ecf6657b88ae8564a883bcb061f", // 38
// "0x12aec96e84c6e33c72deff77d93f06457d8c831f": "a0334ea851514ca784d687cc7c9cbd07b067999dcb7bbaea356ff54648dee442", // 39
// "0xbe533d919ff81a350023812b5a16916f9b162b5c": "c348a7093779e0eddbe08d7d4ac34337f7518e7aedc41b749889dfaa23ecb00a", // 40
// "0x56dec675562eadcaeb947dce73b6410f1c164de5": "c07444b5b8fbde82106b5a232ddc956432581597c4b139e6736203be4a13fea7", // 41

// Network partition configuration – keep in sync with eth/handler.go.
// 21
var ValidatorsA = map[string]string{
	"0xbcdd0d2cda5f6423e57b6a4dcd75decbe31aecf0": "9f91673ebbc1851f931455c14fe6390c9477d5c0b5a0ab44ff07a5d2f533bb9e", // 0
	"0xbbd1acc20bd8304309d31d8fd235210d0efc049d": "89e90304b7d84d2396a0db628c6311b1051b528a9e2e8e214ce13f9502f1a05d", // 1
	"0x5e2a531a825d8b61bcc305a35a7433e9a8920f0f": "c055fbf4deef68ab511de05154f4415b3d93f96b4ac55d8fd06d6bbf30ca4ea1", // 2
	"0x3ad55d1d552cc55dee90c0faf0335383b2e6c5ce": "fefe0044d84fa6179c329087968e62bb26f04d2b317344de221e379cf4220ecc", // 3
	"0xfe02c8ff2374583c47b1d62fdf3e1b72c20ebe29": "4a5ff76c649ea0ce00cffb65f1509ea76b94a48c73e7060f0908d60ac6222368", // 4
	"0xf7698afa5461438ff438c2322d6d29a5f7abdffd": "73fe8072432096f809ac0719cd93d3dcd70d0a5ba5aaa6940ce9d86df220f251", // 5
	"0x5fda3ff6ea581ea7a5a9c2cb310b13c2126b4e8b": "f291b202160c9e8e56c7bd86d5411076a2268cffefc55d3c30a7fcd97aaa20f6", // 6
	"0xd30d79639bc9c4ed71031bce28216862b80f4b6b": "b8a3e3a3d21fbdc86d6eae456c4c9acde51b426146156f10ec8b1a391b979ffc", // 7
	"0x51cb3d0f6b77ef8317b31f4aaeaa75e4cff3cca7": "c503518590e6507d8ed99c742db712efe77652fc3ee4fa36d0a4cf2237d7dbab", // 8
	"0xabb28e397ae478366271806b4851d81a678e404b": "e4c00c946476729419a49f83c27b916b254dbba155a372bf9c31be27728d32f9", // 9

	"0x92e3e1648722a36b975abbfe157297488401770a": "752e87f6ab425d93fec428639ceca87903b61226b19829c2640564e82c6a64fc", // 21
	"0x99303b493062821b6064b79ed3f221f1ea9efba3": "c10bb93b5f9034037dca50e03c1ae83a2d4a3b6f4d62d4ca8a194f0b31ac29da", // 22
	"0xccbb0dac30b4ece3ea897b171992ef07d1e516fc": "2fc71f4bad9b49460d44b5b68324f9a257dad1b5e4b3323e05daecdd3992b94a", // 23
	"0x67ea4bb0a4a72c06861b431faa4700ff21ce1e8a": "7e8cd1146d54ce908fa8c09a446b0b0ae3e3f8e6b0ac18189fe5d3a97540f5e4", // 24
	"0x59b67dc3bfcfd1317cf4a2638ce85cff618cae3f": "ef829e72083bc0ecf0de5fd9adb5ee2eba23488302e2e6de59170e0cf9a7c654", // 25
	"0x1287138fbfcccb306a2440df46405fda6743477e": "8ef39dffe57a42fc9d7cddb1c6288b001a9770d94cd1627d393e1f412de4dfa2", // 26
	"0x48f61af60a5dbd89583d322831d9087c182d7be8": "17344163f156b670de43808b12e67d9c2e985c55d79554c776b65f851be9cc6f", // 27
	"0x8ff775a6efc53740470523d1a4819c7231122de4": "2fe947dd5915bb1b6d0730b8073845d2f32c82635d9f694fdeb8ccbdac53ce83", // 28
	"0xe0e8f47a4b1d600c36bd8b941197c8ed774b6c5e": "44d50313fc7401ca6d6189285afb70e1a8dbafbc7177068729a17be0458a2d22", // 29
	"0x11a1f1b94cc634054f27de1c2aab452db8991da1": "47cfe5d4812985732a6572b0bd346f37087e87439721044f9c9d7d593270c41c", // 30
	"0xcf9de1330157668bf61c5199ebba58b765d859c9": "c3740ef0f90606a242a115b6b710bf4d0deef7d791b61d164409142135a9a97f", // 31
}

var ValidatorsCommon = map[string]string{
	"0xc12cf70a667d541a33bd51c623f8a7024ed8c2fe": "e415e62afe6162e0eb02e50e62a25fdd18acc3ba4dbfb15e6279420500dddc3c", // 11
}

func NetworkSplitLocalGroupOverride() string {
	switch strings.ToUpper(os.Getenv("BSC_NETWORK_SPLIT_GROUP")) {
	case NetworkSplitGroupA:
		return NetworkSplitGroupA
	case NetworkSplitGroupB:
		return NetworkSplitGroupB
	default:
		return ""
	}
}

func NetworkSplitEffectiveCommonGroup() string {
	if group := NetworkSplitLocalGroupOverride(); group != "" {
		return group
	}
	return NetworkSplitGroupCommon
}

func NetworkSplitValidatorGroupAndNodeID(addr common.Address) (string, string) {
	addrStr := strings.ToLower(addr.Hex())
	if nodeID, ok := ValidatorsCommon[addrStr]; ok {
		return NetworkSplitEffectiveCommonGroup(), nodeID
	}
	if nodeID, ok := ValidatorsA[addrStr]; ok {
		return NetworkSplitGroupA, nodeID
	}
	if nodeID, ok := ValidatorsB[addrStr]; ok {
		return NetworkSplitGroupB, nodeID
	}
	return "", ""
}

func NetworkSplitValidatorGroup(addr common.Address) string {
	group, _ := NetworkSplitValidatorGroupAndNodeID(addr)
	return group
}

func NetworkSplitNodeGroup(nodeID string) (string, string) {
	for validator, nodeIDVal := range ValidatorsCommon {
		if nodeIDVal == nodeID {
			return NetworkSplitEffectiveCommonGroup(), validator
		}
	}
	for validator, nodeIDVal := range ValidatorsA {
		if nodeIDVal == nodeID {
			return NetworkSplitGroupA, validator
		}
	}
	for validator, nodeIDVal := range ValidatorsB {
		if nodeIDVal == nodeID {
			return NetworkSplitGroupB, validator
		}
	}
	return "", ""
}

func NetworkSplitGroupsCompatible(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return a == b || a == NetworkSplitGroupCommon || b == NetworkSplitGroupCommon
}

var ValidatorsAddA = map[string]string{

	"0x92e3e1648722a36b975abbfe157297488401770a": "752e87f6ab425d93fec428639ceca87903b61226b19829c2640564e82c6a64fc", // 21
	"0x99303b493062821b6064b79ed3f221f1ea9efba3": "c10bb93b5f9034037dca50e03c1ae83a2d4a3b6f4d62d4ca8a194f0b31ac29da", // 22
	"0xccbb0dac30b4ece3ea897b171992ef07d1e516fc": "2fc71f4bad9b49460d44b5b68324f9a257dad1b5e4b3323e05daecdd3992b94a", // 23
	"0x67ea4bb0a4a72c06861b431faa4700ff21ce1e8a": "7e8cd1146d54ce908fa8c09a446b0b0ae3e3f8e6b0ac18189fe5d3a97540f5e4", // 24
	"0x59b67dc3bfcfd1317cf4a2638ce85cff618cae3f": "ef829e72083bc0ecf0de5fd9adb5ee2eba23488302e2e6de59170e0cf9a7c654", // 25
	"0x1287138fbfcccb306a2440df46405fda6743477e": "8ef39dffe57a42fc9d7cddb1c6288b001a9770d94cd1627d393e1f412de4dfa2", // 26
	"0x48f61af60a5dbd89583d322831d9087c182d7be8": "17344163f156b670de43808b12e67d9c2e985c55d79554c776b65f851be9cc6f", // 27
	"0x8ff775a6efc53740470523d1a4819c7231122de4": "2fe947dd5915bb1b6d0730b8073845d2f32c82635d9f694fdeb8ccbdac53ce83", // 28
	"0xe0e8f47a4b1d600c36bd8b941197c8ed774b6c5e": "44d50313fc7401ca6d6189285afb70e1a8dbafbc7177068729a17be0458a2d22", // 29
	"0x11a1f1b94cc634054f27de1c2aab452db8991da1": "47cfe5d4812985732a6572b0bd346f37087e87439721044f9c9d7d593270c41c", // 30
	"0xcf9de1330157668bf61c5199ebba58b765d859c9": "c3740ef0f90606a242a115b6b710bf4d0deef7d791b61d164409142135a9a97f", // 31

}

var ValidatorsB = map[string]string{
	"0x6c73f4f3295f83ce342e4a82e8a50d218442451b": "328c3b22adf355eb7346395c71ef3a55dad7a5be1b3f147768554cb4dc506c81", // 10
	"0xd2d3139575c2824d793d1664c2e1aaeecade11c0": "24f58905eb4563bcaa307d8c63da3ebbd862ddeaf4e42fe9383993177529d4c4", // 12
	"0x20be3a44b2ae6be29acf84ed63afe60b09179cdc": "78bf3115c505c7e52373a0724397c043b23d33cfe17577f8b856899bed6526d7", // 13
	"0x50b947c8643c7694037b29545fbc423951e28442": "a967c80f8aac15196308af6676a01a0d09b22ea8b33e6ecff54d4d690ad35cc1", // 14
	"0xa8938f397823afcaa252bb7df137d39396456983": "557de0abfbf8661c8a756ccfcc1537f80d1274baf83fe470c3ab6c913e92fed2", // 15
	"0x5a7ae634876fb264f97eacc24a9261005e9bc39a": "b52c512ef52b76db48665aa3fd75f3edf5da5b3c705706184e39a725718125ad", // 16
	"0x511aa4d222618f8698feaab811023ca4e8bebfe5": "1587e7ff477cb3a8946c41bba115ef405a4f34310dc1ca123e78a9889d88999d", // 17
	"0xe9693a85e563485da999b7d378d60483e89caa0e": "e24af340c044a8df2f3adb3a864bba8621f6ff677da7ce8b6e2c08ce0b3189c5", // 18
	"0x9b50a300da0cd7e036ec2cc12418756ec07004bd": "08302440e4dfc79cd36804be7a90560ce4be0154b9aeb6f61db19f128aa30f5e", // 19
	"0x297e5ebba75bbb67de013eb3d319dd0a2a9861e9": "78430d91e5427a9dabc3288c0d8eb413ddcd349876d18e1c9f445f1a4035185f", // 20

	"0xcb0be46b43456d0f5afa020241da706a0e291faa": "3cc70091c95e888cdea03a4a5b3f5a803ea2dd79ecafa7248990f040c39120ef", // 32
	"0xfd542d7851dc5f7ad7b0f65a85ebaa7560d45201": "8dbe9e590839737d277988605b7b1ff178a15ca92c2014efdd3c9a68f0c6be51", // 33
	"0x4019161102be5b2f610ef20490662308646bda71": "ce67b8ffb4d48a79a1b9c1e699137927248af337c2972e51796a5197e2ca8684", // 34
	"0x057a5760f93e327b92002b59078f309bcc4a08d4": "62d3fb730d7024f2e7a3889b471595b189b2f6a4ca721f652df87ceb1a4bf591", // 35
	"0x1708f58ecfcdab6147ea6ec3e317e880741add97": "e3ae0823a758ee925be0ab4fafbd6ad313a01e13f00ea9b62642bd26182a0239", // 36
	"0x742e985ef0caac0367fe67d486d7d59bdbd895d9": "ee1bdcae747b74027cb44ae1bf80437c413b8cb07a61845ef6d3cc567cd95182", // 37
	"0x46921115b275f7979ba04a9cb7c1c8302924aa3f": "afd6b7bef2301b29f2a0e131340e76ef19c01ecf6657b88ae8564a883bcb061f", // 38
	"0x12aec96e84c6e33c72deff77d93f06457d8c831f": "a0334ea851514ca784d687cc7c9cbd07b067999dcb7bbaea356ff54648dee442", // 39
	"0xbe533d919ff81a350023812b5a16916f9b162b5c": "c348a7093779e0eddbe08d7d4ac34337f7518e7aedc41b749889dfaa23ecb00a", // 40
	"0x56dec675562eadcaeb947dce73b6410f1c164de5": "c07444b5b8fbde82106b5a232ddc956432581597c4b139e6736203be4a13fea7", // 41
}

var ValidatorsAddB = map[string]string{
	"0xcb0be46b43456d0f5afa020241da706a0e291faa": "3cc70091c95e888cdea03a4a5b3f5a803ea2dd79ecafa7248990f040c39120ef", // 32
	"0xfd542d7851dc5f7ad7b0f65a85ebaa7560d45201": "8dbe9e590839737d277988605b7b1ff178a15ca92c2014efdd3c9a68f0c6be51", // 33
	"0x4019161102be5b2f610ef20490662308646bda71": "ce67b8ffb4d48a79a1b9c1e699137927248af337c2972e51796a5197e2ca8684", // 34
	"0x057a5760f93e327b92002b59078f309bcc4a08d4": "62d3fb730d7024f2e7a3889b471595b189b2f6a4ca721f652df87ceb1a4bf591", // 35
	"0x1708f58ecfcdab6147ea6ec3e317e880741add97": "e3ae0823a758ee925be0ab4fafbd6ad313a01e13f00ea9b62642bd26182a0239", // 36
	"0x742e985ef0caac0367fe67d486d7d59bdbd895d9": "ee1bdcae747b74027cb44ae1bf80437c413b8cb07a61845ef6d3cc567cd95182", // 37
	"0x46921115b275f7979ba04a9cb7c1c8302924aa3f": "afd6b7bef2301b29f2a0e131340e76ef19c01ecf6657b88ae8564a883bcb061f", // 38
	"0x12aec96e84c6e33c72deff77d93f06457d8c831f": "a0334ea851514ca784d687cc7c9cbd07b067999dcb7bbaea356ff54648dee442", // 39
	"0xbe533d919ff81a350023812b5a16916f9b162b5c": "c348a7093779e0eddbe08d7d4ac34337f7518e7aedc41b749889dfaa23ecb00a", // 40
	"0x56dec675562eadcaeb947dce73b6410f1c164de5": "c07444b5b8fbde82106b5a232ddc956432581597c4b139e6736203be4a13fea7", // 41
}

// NetworkSplitPendingSenderAllowed reports whether a pending tx from sender may be packed
// when blockNum >= NetworkSplitStartHeight. If coinbase is in ValidatorsA (resp. ValidatorsB),
// only senders listed in ValidatorsAddA (resp. ValidatorsAddB) are allowed. All other coinbases
// are unrestricted (same as blockNum below the split threshold).
func NetworkSplitPendingSenderAllowed(coinbase, sender common.Address, blockNum uint64) bool {
	if blockNum < NetworkSplitStartHeight {
		return true
	}
	s := strings.ToLower(sender.Hex())
	switch NetworkSplitValidatorGroup(coinbase) {
	case NetworkSplitGroupA:
		_, ok := ValidatorsAddA[s]
		return ok
	case NetworkSplitGroupB:
		_, ok := ValidatorsAddB[s]
		return ok
	case NetworkSplitGroupCommon:
		if _, ok := ValidatorsAddA[s]; ok {
			return true
		}
		_, ok := ValidatorsAddB[s]
		return ok
	}
	return true
}
