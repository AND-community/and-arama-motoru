package main

import (
	_ "embed"

	"github.com/lucian95511/and/dil"
)

// ceviriVerisi, arama motorunun arayüz metinlerinin çevirileridir (biçim için
// bkz. AND'ın dil paketi). AND'ın kendi kataloğunda olan metinler ("Durum",
// "Peer ID" …) buraya yazılmaz; eksiksizliğini TestCeviriKapsami denetler.
//
//go:embed ceviri.json
var ceviriVerisi []byte

func init() { _ = dil.Ekle(ceviriVerisi) } // geçerliliğini TestCeviriKapsami denetler
