/*
 * Copyright (C) 2021 The Zion Authors
 * This file is part of The Zion library.
 *
 * The Zion is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The Zion is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The Zion.  If not, see <http://www.gnu.org/licenses/>.
 */
package test

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/native"
	scom "github.com/ethereum/go-ethereum/contracts/native/header_sync/common"
	"github.com/ethereum/go-ethereum/contracts/native/header_sync/quorum"
	"github.com/ethereum/go-ethereum/contracts/native/utils"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

var (
	gh = "7b22706172656e7448617368223a22307834623032653838303537336565366165383632623330303638323033366337663363626434323633323666643431643965326566393739633737333637346138222c2273686133556e636c6573223a22307831646363346465386465633735643761616238356235363762366363643431616433313234353162393438613734313366306131343266643430643439333437222c226d696e6572223a22307863313931663630653765333633336634366430313535373530386563383137633461376337323462222c227374617465526f6f74223a22307863316539373733383964613465386637356466343930393232353332343231616439336136623862303439346531376339353136343435656466303561643332222c227472616e73616374696f6e73526f6f74223a22307862363138613633643064346437356261353766303762346233326433323462666661633962303132613133323263613661653738646366363937373335306438222c227265636569707473526f6f74223a22307865376364653339376639326531333339303834646136616338383038626231663537333233623764666163356139356537306130663037316639396534376238222c226c6f6773426c6f6f6d223a2230783030303030303030303030303030303030303030303030303032303030303030303030303030313030323030303030303030303030303030303030303030303030303030303030303030303030303830303030303030303030303030303030303030303030303030303030303030303030303030303038303030303030303030303030303230303030303030313030303030303030303030303030303030303830303030303030303130303830303030303030303030303030303030303030303030303130303030303030303030313030303030383030303030303030303030303030303031303430303030323030303034303030303030303030303030303838303030303030303030303030313030303030303030303030303030303430303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030313030303030303030303030303030303030303030303030303030303030303030303030303230303030303030303030303030303030303030303030303030303030303031303030303030303032303030303030303031303830303030303030303030303030303030303034303030303030303030303030303030303034303030303030303032303030303030303030303030303030222c22646966666963756c7479223a22307831222c226e756d626572223a22307831336662222c226761734c696d6974223a2230783430336566353336222c2267617355736564223a22307830222c2274696d657374616d70223a2230783566653262626539222c22657874726144617461223a22307864393833303130393037383436373635373436383838363736663331326533313335326533333836363436313732373736393665303030303030303030303030663930323832663861383934303366663662656236356665623564613837636131623534363862336539356461373637323535653934323538616634386532386534613638343665393331646466663865316364663835373938323165353934366137303834353563383737373633306161633964316537373032643133663761383635623237633934386330396439333661316234303864366530616661613533376261346530366334353034613061653934616433626635656436343063633732663337626432316436346136356333633735366539633838633934626662353538663064636562303766626230396531633238333034386235353161343331303932313934633039353434383432346135656364356361376363646164666161643132376139643765383865633934643437613465353665393236323534336462333964393230336366316132653533373335663833346238343139613963343938633639333962623935316161646261613531393930663333303635333963346465393965323439373232626339616134383635313634333234366563363966303633636530373465623266653830356239373362646539316463306130386635393538393763376133333737646531386337393562616635353031663930313932623834316530303131623462303434343363333735366337356234336535383837613830613432303166373962306262303832643239363835316236383363623037313835653738646531396362636139616636396466373335373462653030646466643466626339366137353164333164663130396565636339316130363232313861303162383431323433346261313539333134333934303464393737336534366666303537396332323835663762313233663364643638316137646535346463353165626633363735656531633431633030633739386535626238633037366335376333343732653837393839623635303734663966316439663534386661613335643262366530316238343136326461366163663866313230333237306164613535373536343335373035623865633437386335393531383161346132323230323935663437663334636563303364346364326262613135636639353363313731303938313238633439306534326639633834323139323364323232393233306532646337323566383562653030623834313138363263343137633236326161626663373631643334643866636361646165383335386662333437643837633732336331313932623061633631306134383535623138623235653033383663656530306361396431313766323964383863313864656233393535653863313832393865666536663061393962626335663865303062383431353332306233363865376135346163643635383638343338643937636234313865626135666536313536363566653438326238383238356439656466613766323231366534386563623063313332323935353831336435613737616163363533383331656162373335386336323435376566346134666432343164383035346630306238343164326136353936383562326134343136633930353733653563306638656539303562623738316237326436316634353864613366383664663030306130343131343065386361316239356466376538383761306530616639623835643466616235363563316632623831323730303764613235386636623937393435633138613030222c226d697848617368223a22307836333734363936333631366332303632373937613631366537343639366536353230363636313735366337343230373436663663363537323631366536333635222c226e6f6e6365223a22307866666666666666666666666666666666222c2268617368223a22307838623539396139376439303336643633343063323161643730616161333838323539313034323937653666306238623534306537376234383837303835396461227d"
	h1 = "7b22706172656e7448617368223a22307838623539396139376439303336643633343063323161643730616161333838323539313034323937653666306238623534306537376234383837303835396461222c2273686133556e636c6573223a22307831646363346465386465633735643761616238356235363762366363643431616433313234353162393438613734313366306131343266643430643439333437222c226d696e6572223a22307830303030303030303030303030303030303030303030303030303030303030303030303030303030222c227374617465526f6f74223a22307838653539346466323834363764346631663163643034643534613030346535636233313962653539363862666635643432386262383531346334313333396238222c227472616e73616374696f6e73526f6f74223a22307836613963333365653165323562363766313437373536626331613939643435666365373531363562376666323135643933336561653330393564633862363263222c227265636569707473526f6f74223a22307862363434303864613662386665333961623736346166383865636531653863636131633335666439383864623537383036653939313338633632393336356130222c226c6f6773426c6f6f6d223a2230783030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030222c22646966666963756c7479223a22307831222c226e756d626572223a22307831336663222c226761734c696d6974223a2230783430336166313438222c2267617355736564223a22307830222c2274696d657374616d70223a2230783566653262626565222c22657874726144617461223a22307864393833303130393037383436373635373436383838363736663331326533313335326533333836363436313732373736393665303030303030303030303030663930323937663862643934303366663662656236356665623564613837636131623534363862336539356461373637323535653934323538616634386532386534613638343665393331646466663865316364663835373938323165353934366137303834353563383737373633306161633964316537373032643133663761383635623237633934386330396439333661316234303864366530616661613533376261346530366334353034613061653934616433626635656436343063633732663337626432316436346136356333633735366539633838633934626662353538663064636562303766626230396531633238333034386235353161343331303932313934633039353434383432346135656364356361376363646164666161643132376139643765383865633934633139316636306537653336333366343664303135353735303865633831376334613763373234623934643437613465353665393236323534336462333964393230336366316132653533373335663833346238343165363131313835376130383833313038353062633063383139633632373030383436363336666664616638323034326238306530303738663266623735333234323063373166383933386130643465383164333539383736313965393265393064313634336465616365646533623464616563356536343839613161613135333031663930313932623834316266663662366161306361643738336534396336323763316365333264366336643064393937653763353430646239323766666435306233653364653163333434666132333964353434663338393239303463623764396631363565616239653561633633336432363161613238396631393639333964376361653938643161303062383431373135333539626565323232346666373764316538663433306362303566343965643631643164303364633163313464643161396166653462316132383262383539383936613561333463363934613965653562653439656535303832363334636238346334643438663963653039643763396366333662363738313932323730316238343136653430643766326463633264616433623636373338333130326635356163383032353731313563373162623533316564643334363865363034313763613332343437656661306565613437336531623236363135653237666230323037343739306264343863613838323665316661653863353764306533663235333937373031623834313862383230366631306530333638663431666264353731306531383235643137306339653465333761353861303164336335353331653435313732633135663631653435636335656562383030346262623962363966393436383631623861636265623434323065366631666334636533356530396565346136643637613139303162383431383935316266313632656630306534353465313933313665643261613537373035386536653366643632643964613965626432663861303939343235613930613464393034333566633837666562383665616662353330626434393162666462636137623332393434343037396431666235663239653561626262343936326630316238343130303836386337366262346332643138353932333838613566663763623965303262393031343961306235303739663631393562613134633730366435313531303462326161393237393834613662346163303738333833663137616639373761313965623932663562363666613762353465353636643961343030316638613031222c226d697848617368223a22307836333734363936333631366332303632373937613631366537343639366536353230363636313735366337343230373436663663363537323631366536333635222c226e6f6e6365223a22307830303030303030303030303030303030222c2268617368223a22307831626137396532353231313833616532633135633364326563336436623366656137633438613337333232326135623164386530663664386564363737343231227d"
	h2 = "7b22706172656e7448617368223a22307833346333633638323238366364653563346133313565623731643337656534383536343937363763333535333164663264636234363134386636383130386365222c2273686133556e636c6573223a22307831646363346465386465633735643761616238356235363762366363643431616433313234353162393438613734313366306131343266643430643439333437222c226d696e6572223a22307830303030303030303030303030303030303030303030303030303030303030303030303030303030222c227374617465526f6f74223a22307835653436613162636162653138363633613638303533656363633733393132313339396233393931613061313335613065363531373236333933656435333135222c227472616e73616374696f6e73526f6f74223a22307863356230623838363638313063323665303665623838343435643866643733646537346637643434353139393461653237376434343631303766306166356533222c227265636569707473526f6f74223a22307862363434303864613662386665333961623736346166383865636531653863636131633335666439383864623537383036653939313338633632393336356130222c226c6f6773426c6f6f6d223a2230783030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030222c22646966666963756c7479223a22307831222c226e756d626572223a22307831343161222c226761734c696d6974223a2230783366633265666434222c2267617355736564223a22307830222c2274696d657374616d70223a2230783566653262633834222c22657874726144617461223a22307864393833303130393037383436373635373436383838363736663331326533313335326533333836363436313732373736393665303030303030303030303030663930323832663861383934303366663662656236356665623564613837636131623534363862336539356461373637323535653934323538616634386532386534613638343665393331646466663865316364663835373938323165353934366137303834353563383737373633306161633964316537373032643133663761383635623237633934386330396439333661316234303864366530616661613533376261346530366334353034613061653934616433626635656436343063633732663337626432316436346136356333633735366539633838633934626662353538663064636562303766626230396531633238333034386235353161343331303932313934633039353434383432346135656364356361376363646164666161643132376139643765383865633934643437613465353665393236323534336462333964393230336366316132653533373335663833346238343135376165653833313331623663366335653438396661393761396432666135353763386232613132633963613563313238636432613166313963336465373631366431646138646632616366396438663161383436363932656364623533366363653765306432626162383335656662626434643865356335653433353239353031663930313932623834316537373930643537663735633562623137356663396333333366353630646230373861626335633238316236316238323630653330666365313263383865353531653637653531393633386362613462636366623664383065323462616538353836663237353761393936353131386161376530373131306130633231373563303062383431333333393134623030383533643833656166383139313839336232333239643837643639363862363633393232303965613338626539623831316161393232363433316364636132623766663563383938646165303335333233613237353834336539356332356431366538353437653430656266326136363265323736623030316238343131613563666333353735333435646463343533643363633834366665666234363538623333343136353464616166633233313232646365346535643963656237346432636165316363386239326339393166643534353563613436663864313339373530363433663436623134353234336365386530363864323833613365383031623834316632346236626162383566383266356565326331373962366461363031326335663566333637646430313531636235646138386333353235363931343938396636373433396535613339653732656262643435363163653331396161626364353737373731616239646364643435643661666337393262383535313165313531303162383431373936656462633061636237646236323637666465383536313866306435303936346431376630363832333964383362653930613936623661623436306139613338326662326433393938613437396532653839313739373762333838346531323262313638386530646639356561383566646133356335333335393036653430316238343137613664383364383731376131626139633363626566316132333666623965323237343730336231366333366564623439343638353736366639343664363063306130313766363939646461306130616433303263376138646337326633656332333461623638636635613165663866393533653635323630353935326661663030222c226d697848617368223a22307836333734363936333631366332303632373937613631366537343639366536353230363636313735366337343230373436663663363537323631366536333635222c226e6f6e6365223a22307830303030303030303030303030303030222c2268617368223a22307832643031336334633036343766653831303838343031316664656365633830633135383761646433636166383831396263613065316537363534323762646130227d"
)

func TestQuorumHandler_SyncGenesisHeader(t *testing.T) {
	raw, err := hex.DecodeString(gh)
	if err != nil {
		t.Fatal(err)
	}
	param := &scom.SyncGenesisHeaderParam{
		ChainID:       quorumChainID,
		GenesisHeader: raw,
	}

	input, err := utils.PackMethodWithStruct(scom.ABI, scom.MethodSyncGenesisHeader, param)
	assert.Nil(t, err)

	caller := crypto.PubkeyToAddress(*acct)
	blockNumber := big.NewInt(1)
	extra := uint64(10)
	contractRef := native.NewContractRef(sdb, caller, caller, blockNumber, common.Hash{}, scom.GasTable[scom.MethodSyncGenesisHeader]+extra, nil)
	ret, leftOverGas, err := contractRef.NativeCall(caller, utils.HeaderSyncContractAddress, input)

	assert.Nil(t, err)

	result, err := utils.PackOutputs(scom.ABI, scom.MethodSyncGenesisHeader, true)
	assert.Nil(t, err)
	assert.Equal(t, ret, result)
	assert.Equal(t, leftOverGas, extra)

	contract := native.NewNativeContract(sdb, contractRef)
	pvs, err := quorum.GetValSet(contract, quorumChainID)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(pvs)
}

func TestQuorumHandler_SyncBlockHeader(t *testing.T) {
	raw, err := hex.DecodeString(gh)
	if err != nil {
		t.Fatal(err)
	}
	param := &scom.SyncGenesisHeaderParam{
		ChainID:       quorumChainID,
		GenesisHeader: raw,
	}

	input, err := utils.PackMethodWithStruct(scom.ABI, scom.MethodSyncGenesisHeader, param)
	assert.Nil(t, err)

	caller := crypto.PubkeyToAddress(*acct)
	blockNumber := big.NewInt(1)
	extra := uint64(10)
	contractRef := native.NewContractRef(sdb, caller, caller, blockNumber, common.Hash{}, scom.GasTable[scom.MethodSyncGenesisHeader]+extra, nil)
	ret, leftOverGas, err := contractRef.NativeCall(caller, utils.HeaderSyncContractAddress, input)

	assert.Nil(t, err)

	result, err := utils.PackOutputs(scom.ABI, scom.MethodSyncGenesisHeader, true)
	assert.Nil(t, err)
	assert.Equal(t, ret, result)
	assert.Equal(t, leftOverGas, extra)

	contract := native.NewNativeContract(sdb, contractRef)

	pvs, err := quorum.GetValSet(contract, quorumChainID)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(pvs)

	{
		raw, err = hex.DecodeString(h1)
		if err != nil {
			t.Fatal(err)
		}
		p1 := &scom.SyncBlockHeaderParam{
			ChainID: quorumChainID,
			Address: caller,
			Headers: [][]byte{
				raw,
			},
		}
		input, err = utils.PackMethodWithStruct(scom.ABI, scom.MethodSyncBlockHeader, p1)
		assert.Nil(t, err)

		contractRef = native.NewContractRef(sdb, caller, caller, blockNumber, common.Hash{}, scom.GasTable[scom.MethodSyncBlockHeader]+extra, nil)
		ret, leftOverGas, err = contractRef.NativeCall(caller, utils.HeaderSyncContractAddress, input)

		assert.Nil(t, err)

		result, err = utils.PackOutputs(scom.ABI, scom.MethodSyncBlockHeader, true)
		assert.Nil(t, err)
		assert.Equal(t, ret, result)
		assert.Equal(t, leftOverGas, extra)

		pvs, err = quorum.GetValSet(contract, quorumChainID)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(pvs)
	}

	{
		raw, err = hex.DecodeString(h2)
		if err != nil {
			t.Fatal(err)
		}
		p2 := &scom.SyncBlockHeaderParam{
			ChainID: quorumChainID,
			Address: caller,
			Headers: [][]byte{
				raw,
			},
		}

		input, err = utils.PackMethodWithStruct(scom.ABI, scom.MethodSyncBlockHeader, p2)
		assert.Nil(t, err)

		contractRef = native.NewContractRef(sdb, caller, caller, blockNumber, common.Hash{}, scom.GasTable[scom.MethodSyncBlockHeader]+extra, nil)
		ret, leftOverGas, err = contractRef.NativeCall(caller, utils.HeaderSyncContractAddress, input)

		assert.Nil(t, err)

		result, err = utils.PackOutputs(scom.ABI, scom.MethodSyncBlockHeader, true)
		assert.Nil(t, err)
		assert.Equal(t, ret, result)
		assert.Equal(t, leftOverGas, extra)

		pvs, err = quorum.GetValSet(contract, quorumChainID)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(pvs)
	}

}
