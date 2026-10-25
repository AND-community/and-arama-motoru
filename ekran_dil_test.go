package main

// ekran_dil_test.go — her ekran, desteklenen her dilde ve iki terminal
// boyutunda çizilir: çevrilmeden kalmış Türkçe metin olmamalı ve hiçbir satır
// terminal genişliğini aşmamalı. ARAMA_EKRAN_DOKUM=<dizin> verilirse ekranlar
// göz ile incelemek için düz metin olarak o dizine yazılır.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/lucian95511/and/dil/diltest"
)

// ekranOrnekleri, ekranlara konan içeriktir. Kullanıcı içeriği Türkçe harf
// içermez (denetçi onu kalıntı sayardı); kategori ise ağdaki gibi Türkçedir
// ve ekranda çevrilmiş görünmelidir.
func ekranOrnekleri() []AramaSonucu {
	zaman := time.Date(2026, 10, 20, 14, 30, 0, 0, time.UTC)
	yazar := strings.Repeat("ab", 32)
	return []AramaSonucu{
		{Tip: TipKonu, KaynakID: "k1", Kategori: "Yapay Zeka", Baslik: "Hello world", Govde: "first post https://example.com/page", YazarAd: "ali", YazarKey: yazar, OlusturmaTS: zaman},
		{Tip: TipYanit, KaynakID: "y1", UstID: "k1", Govde: "a reply", YazarAd: "can", YazarKey: strings.Repeat("cd", 32), OlusturmaTS: zaman},
		{Tip: TipSohbet, KaynakID: "s1", Govde: "chat message", YazarAd: "deniz", YazarKey: strings.Repeat("ef", 32), OlusturmaTS: zaman},
		{Tip: TipSite, KaynakID: yazar + "/about", Baslik: "About", Govde: "site body and-site://" + yazar + "/other", YazarAd: "ali", YazarKey: yazar, OlusturmaTS: zaman},
	}
}

func TestEkranlarHerDilde(t *testing.T) {
	denetci := diltest.YeniDenetci(t, nil, diltest.Oku(t, "ceviri.json", ceviriVerisi))
	dokum := os.Getenv("ARAMA_EKRAN_DOKUM")
	ornekler := ekranOrnekleri()

	diltest.HerDilde(t, func(kod string) {
		// Yer tutucular model kurulurken çevrildiği için model her dilde yeniden kurulur.
		taban := testModel(t)
		for _, s := range ornekler {
			if err := taban.depo.Ekle(s); err != nil {
				t.Fatal(err)
			}
		}
		if err := taban.depo.TakipEkle("hello", TipKonu, ""); err != nil {
			t.Fatal(err)
		}
		takipler, err := taban.depo.TakipListele()
		if err != nil {
			t.Fatal(err)
		}
		taban.indeksSayisi, taban.peerSayisi = len(ornekler), 2

		ekranlar := []struct {
			ad  string
			kur func(m model) model
		}{
			{"anamenu", func(m model) model { m.ekran = ekAnaMenu; return m }},
			{"ara-bos", func(m model) model { m.ekran = ekArama; return m }},
			{"ara", func(m model) model {
				m.ekran = ekArama
				m.aramaIn.SetValue("hello")
				m.yerelSonuclar = ornekler[:3]
				ag := ornekler[3]
				ag.KaynakPeer = "12D3KooWExamplePeer"
				m.agSonuclari = []AramaSonucu{ag}
				return m
			}},
			{"detay-konu", func(m model) model { m.ekran = ekDetay; m.seciliSonuc = &ornekler[0]; return m }},
			{"detay-site", func(m model) model {
				m.ekran = ekDetay
				ag := ornekler[3]
				ag.KaynakPeer = "12D3KooWExamplePeer"
				m.seciliSonuc = &ag
				return m
			}},
			{"durum", func(m model) model { m.ekran = ekDurum; return m }},
			{"kodlagit", func(m model) model { m.ekran = ekKodlaGit; return m }},
			{"yardim", func(m model) model { m.ekran = ekYardim; return m }},
			{"takip", func(m model) model { m.ekran = ekTakip; m.takipListesi = takipler; return m }},
		}
		for _, boyut := range [][2]int{{80, 24}, {120, 40}} {
			next, _ := taban.Update(tea.WindowSizeMsg{Width: boyut[0], Height: boyut[1]})
			m := next.(model)
			for _, e := range ekranlar {
				ekran := ansi.Strip(e.kur(m).View())
				yer := fmt.Sprintf("%s %dx%d %s", kod, boyut[0], boyut[1], e.ad)
				for _, k := range denetci.Kalintilar(kod, ekran) {
					t.Errorf("%s: %s", yer, k)
				}
				for i, satir := range strings.Split(ekran, "\n") {
					if g := lipgloss.Width(satir); g > boyut[0] {
						t.Errorf("%s: %d. satır %d sütun (> %d): %q", yer, i+1, g, boyut[0], satir)
					}
					// Çerçeve sığmayan satırı içerik kenarında "…" ile kırpar.
					if strings.HasSuffix(satir, "…  │") {
						t.Errorf("%s: %d. satır kırpılmış: %q", yer, i+1, satir)
					}
				}
				if dokum != "" {
					ad := fmt.Sprintf("%s_%dx%d_%s.txt", kod, boyut[0], boyut[1], e.ad)
					if err := os.WriteFile(filepath.Join(dokum, ad), []byte(ekran), 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
	})
}
