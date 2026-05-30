package params

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// Attack 2 Experiment layout.

// At block height 397, a block is produced by address 0xf7... (this follows Parlia's natural in-turn schedule).
// Starting from height 398, we enter the experiment window.
// During the experiment, nodes no longer broadcast blocks freely; instead, they only send blocks via directed delivery
// according to the routing table below: (height, miner) → target validator.

// Two parallel chains:

// ```text
//   difficulty :            2        →    1        →    2        →    1        →    2        →    1        →    2        →    1        →    2        →    1        →    2        →    1
//   Chain A (no prime):  398-399(5e) → 400-407(29) → 408-415(6c) → 416-423(50) → 424-431(a8) → 432-439(51) → 440-447(bc) → 448-455(20) → 456-463(d2) → 464-471(5f) → 472-479(e9) → 480-487(9b)
// ```

// ```text
//   difficulty :            1          →    2          →    1          →    2          →    1          →     2         →    1          →    2          →    1          →    2          →    1          →    2
//   Chain B (prime):     398'-399'(20) → 400'-407'(5f) → 408'-415'(3a) → 416'-423'(9b) → 424'-431'(511) → 432'-439'(bb) → 440'-447'(5a) → 448'-455'(c1) → 456'-463'(fe) → 464'-471'(d3) → 472'-479'(29) → 480'-487'(f7)
// ```

// ```

const (
	// FirstUpdateValidatorSetHeight / SecondUpdateValidatorSetHeight 保留原有语义。
	FirstUpdateValidatorSetHeight  uint64 = 199
	SecondUpdateValidatorSetHeight uint64 = 399

	// 实验窗口 [NetworkSplitStartHeight, NetworkSplitEndHeight)。
	// NetworkSplitStartHeight uint64 = 398
	// NetworkSplitEndHeight   uint64 = 620
	// NetworkSplitManualEnd   uint64 = 490

	NetworkSplitStartHeight uint64 = 398
	NetworkSplitEndHeight   uint64 = 1000
	NetworkSplitManualEnd   uint64 = 487

	NetworkSplitGroupA = "A"
	NetworkSplitGroupB = "B"
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

// 参与实验的全部验证人（地址 → enode ID）。
var AllValidators = map[string]string{
	"0xbcdd0d2cda5f6423e57b6a4dcd75decbe31aecf0": "9f91673ebbc1851f931455c14fe6390c9477d5c0b5a0ab44ff07a5d2f533bb9e", // bc
	"0xbbd1acc20bd8304309d31d8fd235210d0efc049d": "89e90304b7d84d2396a0db628c6311b1051b528a9e2e8e214ce13f9502f1a05d", // bb
	"0xf7698afa5461438ff438c2322d6d29a5f7abdffd": "73fe8072432096f809ac0719cd93d3dcd70d0a5ba5aaa6940ce9d86df220f251", // f7
	"0x5fda3ff6ea581ea7a5a9c2cb310b13c2126b4e8b": "f291b202160c9e8e56c7bd86d5411076a2268cffefc55d3c30a7fcd97aaa20f6", // 5f
	"0x5e2a531a825d8b61bcc305a35a7433e9a8920f0f": "c055fbf4deef68ab511de05154f4415b3d93f96b4ac55d8fd06d6bbf30ca4ea1", // 5e
	"0xfe02c8ff2374583c47b1d62fdf3e1b72c20ebe29": "4a5ff76c649ea0ce00cffb65f1509ea76b94a48c73e7060f0908d60ac6222368", // fe
	"0x3ad55d1d552cc55dee90c0faf0335383b2e6c5ce": "fefe0044d84fa6179c329087968e62bb26f04d2b317344de221e379cf4220ecc", // 3a
	"0xd30d79639bc9c4ed71031bce28216862b80f4b6b": "b8a3e3a3d21fbdc86d6eae456c4c9acde51b426146156f10ec8b1a391b979ffc", // d3
	"0x51cb3d0f6b77ef8317b31f4aaeaa75e4cff3cca7": "c503518590e6507d8ed99c742db712efe77652fc3ee4fa36d0a4cf2237d7dbab", // 51
	"0xabb28e397ae478366271806b4851d81a678e404b": "e4c00c946476729419a49f83c27b916b254dbba155a372bf9c31be27728d32f9", // ab
	"0x6c73f4f3295f83ce342e4a82e8a50d218442451b": "328c3b22adf355eb7346395c71ef3a55dad7a5be1b3f147768554cb4dc506c81", // 6c
	"0xc12cf70a667d541a33bd51c623f8a7024ed8c2fe": "e415e62afe6162e0eb02e50e62a25fdd18acc3ba4dbfb15e6279420500dddc3c", // c1
	"0xd2d3139575c2824d793d1664c2e1aaeecade11c0": "24f58905eb4563bcaa307d8c63da3ebbd862ddeaf4e42fe9383993177529d4c4", // d2
	"0x20be3a44b2ae6be29acf84ed63afe60b09179cdc": "78bf3115c505c7e52373a0724397c043b23d33cfe17577f8b856899bed6526d7", // 20
	"0x50b947c8643c7694037b29545fbc423951e28442": "a967c80f8aac15196308af6676a01a0d09b22ea8b33e6ecff54d4d690ad35cc1", // 50
	"0xa8938f397823afcaa252bb7df137d39396456983": "557de0abfbf8661c8a756ccfcc1537f80d1274baf83fe470c3ab6c913e92fed2", // a8
	"0x5a7ae634876fb264f97eacc24a9261005e9bc39a": "b52c512ef52b76db48665aa3fd75f3edf5da5b3c705706184e39a725718125ad", // 5a
	"0x511aa4d222618f8698feaab811023ca4e8bebfe5": "1587e7ff477cb3a8946c41bba115ef405a4f34310dc1ca123e78a9889d88999d", // 511
	"0xe9693a85e563485da999b7d378d60483e89caa0e": "e24af340c044a8df2f3adb3a864bba8621f6ff677da7ce8b6e2c08ce0b3189c5", // e9
	"0x9b50a300da0cd7e036ec2cc12418756ec07004bd": "08302440e4dfc79cd36804be7a90560ce4be0154b9aeb6f61db19f128aa30f5e", // 9b
	"0x297e5ebba75bbb67de013eb3d319dd0a2a9861e9": "78430d91e5427a9dabc3288c0d8eb413ddcd349876d18e1c9f445f1a4035185f", // 29
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

// 实验中参与出块的验证人地址（全部小写，带 0x 前缀）。
const (
	expAddr20  = "0x20be3a44b2ae6be29acf84ed63afe60b09179cdc"
	expAddr29  = "0x297e5ebba75bbb67de013eb3d319dd0a2a9861e9"
	expAddr3a  = "0x3ad55d1d552cc55dee90c0faf0335383b2e6c5ce"
	expAddr50  = "0x50b947c8643c7694037b29545fbc423951e28442"
	expAddr511 = "0x511aa4d222618f8698feaab811023ca4e8bebfe5"
	expAddr51c = "0x51cb3d0f6b77ef8317b31f4aaeaa75e4cff3cca7"
	expAddr5a  = "0x5a7ae634876fb264f97eacc24a9261005e9bc39a"
	expAddr5e  = "0x5e2a531a825d8b61bcc305a35a7433e9a8920f0f"
	expAddr5f  = "0x5fda3ff6ea581ea7a5a9c2cb310b13c2126b4e8b"
	expAddr6c  = "0x6c73f4f3295f83ce342e4a82e8a50d218442451b"
	expAddr9b  = "0x9b50a300da0cd7e036ec2cc12418756ec07004bd"
	expAddrA8  = "0xa8938f397823afcaa252bb7df137d39396456983"
	expAddrAbb = "0xabb28e397ae478366271806b4851d81a678e404b"
	expAddrBb  = "0xbbd1acc20bd8304309d31d8fd235210d0efc049d"
	expAddrBc  = "0xbcdd0d2cda5f6423e57b6a4dcd75decbe31aecf0"
	expAddrC12 = "0xc12cf70a667d541a33bd51c623f8a7024ed8c2fe"
	expAddrD2  = "0xd2d3139575c2824d793d1664c2e1aaeecade11c0"
	expAddrD30 = "0xd30d79639bc9c4ed71031bce28216862b80f4b6b"
	expAddrE9  = "0xe9693a85e563485da999b7d378d60483e89caa0e"
	expAddrF7  = "0xf7698afa5461438ff438c2322d6d29a5f7abdffd"
	expAddrFe  = "0xfe02c8ff2374583c47b1d62fdf3e1b72c20ebe29"
)

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

// ExperimentSlot 描述一个实验调度单元：height 高度上某个矿工出块，并只允许广播给 Targets 列表中的验证人。
// Targets 为空表示不广播（只保留在本地）。
type ExperimentSlot struct {
	Miner   common.Address
	Targets []common.Address
	Branch  string
}

// experimentSchedule 按 (高度, 矿工地址 lower) 组织，便于 O(1) 查找广播目标。
var experimentSchedule = buildExperimentSchedule()

// experimentMiners 按高度聚合所有允许出块的矿工（用于 Seal 门禁）。
var experimentMiners = buildExperimentMiners()

type scheduleKey struct {
	height uint64
	miner  string // lowercase hex address
}

func appendTargets(targets []common.Address, addrMaps ...map[string]string) []common.Address {
	seen := make(map[common.Address]struct{}, len(targets))
	for _, target := range targets {
		seen[target] = struct{}{}
	}
	for _, addrMap := range addrMaps {
		for addr := range addrMap {
			target := common.HexToAddress(addr)
			if _, ok := seen[target]; ok {
				continue
			}
			targets = append(targets, target)
			seen[target] = struct{}{}
		}
	}
	return targets
}

func appendTargetList(targets []common.Address, targetList []string) []common.Address {
	for _, target := range targetList {
		targets = append(targets, common.HexToAddress(target))
	}
	return targets
}

func appendAddressList(targets []common.Address, targetList []string) []common.Address {
	seen := make(map[common.Address]struct{}, len(targets))
	for _, target := range targets {
		seen[target] = struct{}{}
	}
	for _, targetHex := range targetList {
		target := common.HexToAddress(targetHex)
		if _, ok := seen[target]; ok {
			continue
		}
		targets = append(targets, target)
		seen[target] = struct{}{}
	}
	return targets
}

// idx=3 addr=0x297E5eBBa75BBb67de013eB3D319Dd0A2a9861E9 grp=?;
//  idx=6 addr=0x50B947C8643C7694037b29545fbc423951e28442 grp=?;
//   idx=7 addr=0x511Aa4D222618F8698fEaaB811023Ca4E8Bebfe5 grp=?;
//    idx=8 addr=0x51cb3d0f6b77ef8317b31f4AaEaa75E4cfF3CCa7 grp=?;
//    idx=10 addr=0x5A7ae634876Fb264f97EaCC24a9261005e9bc39a grp=?;
//    idx=11 addr=0x5e2A531A825d8B61BCc305a35a7433e9A8920f0f grp=?;
//    idx=16 addr=0x9B50A300Da0cD7E036EC2cC12418756Ec07004BD

var after487LegacyTargetsA = []string{
	expAddr50,
	expAddr9b,
	expAddrE9,
	expAddrD2,
	expAddr20,
	expAddrBc,
	expAddr5f,
}

// idx=2 addr=0x1708f58EcfCDAb6147eA6EC3E317E880741Add97 grp=?;
// idx=3 addr=0x20Be3a44B2Ae6Be29aCF84ED63aFe60b09179CdC grp=?;
// idx=4 addr=0x297E5eBBa75BBb67de013eB3D319Dd0A2a9861E9 grp=?;
// idx=5 addr=0x3aD55d1D552cC55dee90C0faf0335383B2e6C5cE grp=?;
// idx=8 addr=0x50B947C8643C7694037b29545fbc423951e28442 grp=?;
// idx=10 addr=0x51cb3d0f6b77ef8317b31f4AaEaa75E4cfF3CCa7 grp=?;
//  idx=14 addr=0x5FDA3Ff6ea581EA7A5a9C2CB310b13c2126b4e8B grp=?;
//  idx=15 addr=0x6C73f4f3295F83cE342e4A82e8a50D218442451b grp=?;
//  idx=17 addr=0xbCDD0d2cDA5f6423E57B6a4dCD75DEcbE31aeCf0 grp=?

var after487LegacyTargetsB = []string{
	expAddr29,
	expAddr3a,
	expAddrF7,
	expAddrD30,
	expAddrFe,
	expAddrC12,
	expAddr5a,
	expAddrBb,
	expAddr511,
}

func heights(lo, hi uint64) []uint64 {
	out := make([]uint64, 0, hi-lo+1)
	for h := lo; h <= hi; h++ {
		out = append(out, h)
	}
	return out
}

func buildExperimentSchedule() map[scheduleKey]ExperimentSlot {
	rows := []struct {
		height     []uint64
		miner      string
		targetList []string
	}{
		{heights(398, 399), expAddr5e, []string{expAddr29}},
		{heights(398, 399), expAddr20, []string{expAddr5f}},

		{heights(400, 407), expAddr29, []string{expAddr6c}},
		{heights(400, 407), expAddr5f, []string{expAddr3a}},

		{heights(408, 415), expAddr6c, []string{expAddr50}},
		{heights(408, 415), expAddr3a, []string{expAddr9b}},

		{heights(416, 423), expAddr50, []string{expAddrA8}},
		{heights(416, 423), expAddr9b, []string{expAddr511}},

		{heights(424, 431), expAddrA8, []string{expAddr51c}},
		{heights(424, 431), expAddr511, []string{expAddrBb}},

		{heights(432, 439), expAddr51c, []string{expAddrBc}},
		{heights(432, 439), expAddrBb, []string{expAddr5a}},

		{heights(440, 447), expAddrBc, []string{expAddr20}},
		{heights(440, 447), expAddr5a, []string{expAddrC12}},

		{heights(448, 455), expAddr20, []string{expAddrD2}},
		{heights(448, 455), expAddrC12, []string{expAddrFe}},

		{heights(456, 463), expAddrD2, []string{expAddr5f}},
		{heights(456, 463), expAddrFe, []string{expAddrD30}},

		{heights(464, 471), expAddr5f, []string{expAddrE9}},
		{heights(464, 471), expAddrD30, []string{expAddr29}},

		{heights(472, 479), expAddrE9, []string{expAddr9b}},
		{heights(472, 479), expAddr29, []string{expAddrF7}},

		{heights(480, 487), expAddr9b, []string{expAddrE9}},
		{heights(480, 487), expAddrF7, []string{expAddr29}},
	}
	m := make(map[scheduleKey]ExperimentSlot, len(rows)*8)
	for idx, r := range rows {
		branch := NetworkSplitGroupA
		if idx%2 == 1 {
			branch = NetworkSplitGroupB
		}
		minerLower := strings.ToLower(r.miner)
		for _, h := range r.height {
			targets := appendTargetList(make([]common.Address, 0, len(r.targetList)), r.targetList)
			if h >= NetworkSplitManualEnd {
				if branch == NetworkSplitGroupA {
					targets = appendTargets(targets, ValidatorsAddA)
					targets = appendAddressList(targets, after487LegacyTargetsA)
				} else {
					targets = appendTargets(targets, ValidatorsAddB)
					targets = appendAddressList(targets, after487LegacyTargetsB)
				}
			}
			m[scheduleKey{height: h, miner: minerLower}] = ExperimentSlot{
				Miner:   common.HexToAddress(r.miner),
				Targets: targets,
				Branch:  branch,
			}
		}
	}
	return m
}

func buildExperimentMiners() map[uint64]map[common.Address]struct{} {
	m := make(map[uint64]map[common.Address]struct{})
	for k := range experimentSchedule {
		set, ok := m[k.height]
		if !ok {
			set = make(map[common.Address]struct{})
			m[k.height] = set
		}
		set[common.HexToAddress(k.miner)] = struct{}{}
	}
	return m
}

func branchForLateExperimentMiner(miner common.Address) string {
	addr := strings.ToLower(miner.Hex())
	if _, ok := ValidatorsAddA[addr]; ok {
		return NetworkSplitGroupA
	}
	if _, ok := ValidatorsAddB[addr]; ok {
		return NetworkSplitGroupB
	}
	for _, target := range after487LegacyTargetsA {
		if addr == target {
			return NetworkSplitGroupA
		}
	}
	for _, target := range after487LegacyTargetsB {
		if addr == target {
			return NetworkSplitGroupB
		}
	}
	return ""
}

// IsExperimentActive reports whether height falls within the experiment window [398, NetworkSplitEndHeight).
func IsExperimentActive(height uint64) bool {
	return height >= NetworkSplitStartHeight && height < NetworkSplitEndHeight
}

// CanMineExperimentBlock reports whether addr is allowed to seal a block at the given height.
// Before NetworkSplitManualEnd, only validators listed in experimentSchedule at that height may seal.
// From NetworkSplitManualEnd onward, validators seal naturally while block propagation remains partitioned.
func CanMineExperimentBlock(height uint64, addr common.Address) bool {
	if height < NetworkSplitStartHeight {
		return true
	}
	if height >= NetworkSplitManualEnd {
		return true
	}
	set, ok := experimentMiners[height]
	if !ok {
		return false
	}
	_, ok = set[addr]
	return ok
}

// ExperimentBroadcastTargets returns the peer validator addresses to which the block
// (height, miner) should be broadcast during the experiment.
//
// Return values:
//
//	inWindow == false: height is outside the experiment window; caller should broadcast normally.
//	inWindow == true,  targets == nil: the (height, miner) pair is unknown for this partition — drop to be safe.
//	inWindow == true,  targets != nil: send only to the listed validator peers.
func ExperimentBroadcastTargets(height uint64, miner common.Address) (targets []common.Address, inWindow bool) {
	if !IsExperimentActive(height) {
		return nil, false
	}
	if height >= NetworkSplitManualEnd {
		switch branchForLateExperimentMiner(miner) {
		case NetworkSplitGroupA:
			targets = appendTargets(nil, ValidatorsAddA)
			targets = appendAddressList(targets, after487LegacyTargetsA)
			return targets, true
		case NetworkSplitGroupB:
			targets = appendTargets(nil, ValidatorsAddB)
			targets = appendAddressList(targets, after487LegacyTargetsB)
			return targets, true
		default:
			return nil, true
		}
	}
	key := scheduleKey{height: height, miner: strings.ToLower(miner.Hex())}
	slot, ok := experimentSchedule[key]
	if !ok {
		// In-window block from a miner NOT in the experiment schedule – drop silently.
		return nil, true
	}
	return slot.Targets, true
}

func NetworkSplitValidatorGroup(coinbase common.Address, blockNum uint64) string {
	if blockNum >= NetworkSplitManualEnd {
		return branchForLateExperimentMiner(coinbase)
	}
	slot, ok := experimentSchedule[scheduleKey{
		height: blockNum,
		miner:  strings.ToLower(coinbase.Hex()),
	}]
	if !ok {
		return ""
	}
	return slot.Branch
}

// NetworkSplitPendingSenderAllowed reports whether a pending tx from sender may be packed
// when blockNum >= NetworkSplitStartHeight. For attack 2, branch membership is schedule-based:
// e.g. at height 399, 6c is the A-chain miner and 20 is the B-chain miner.
func NetworkSplitPendingSenderAllowed(coinbase, sender common.Address, blockNum uint64) bool {
	if blockNum < NetworkSplitStartHeight {
		return true
	}
	s := strings.ToLower(sender.Hex())
	switch NetworkSplitValidatorGroup(coinbase, blockNum) {
	case NetworkSplitGroupA:
		_, ok := ValidatorsAddA[s]
		return ok
	case NetworkSplitGroupB:
		_, ok := ValidatorsAddB[s]
		return ok
	}
	return true
}
