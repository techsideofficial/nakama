package evr

import (
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/samber/lo"
)

func TestLobbyFindSessionRequest_Unmarshal(t *testing.T) {
	data := []byte{
		0xf6, 0x40, 0xbb, 0x78, 0xa2, 0xe7, 0x8c, 0xbb,
		0xf5, 0xa3, 0x9a, 0x81, 0x01, 0x2a, 0x2c, 0x31,
		0xa7, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x0d, 0x91, 0x77, 0x8f, 0xd7, 0x01, 0x2f, 0xc6,
		0x73, 0xaf, 0x1c, 0x7e, 0xde, 0xa4, 0x60, 0xcb,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xe5, 0xef, 0x94, 0xa8, 0xb1, 0xd0, 0xe8, 0xc8,
		0xf0, 0x3c, 0x22, 0x0a, 0x73, 0x8b, 0xfa, 0x3d,
		0x2e, 0xe8, 0x77, 0x74, 0xc3, 0xb2, 0x80, 0x97,
		0x01, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x7b, 0x22, 0x67, 0x61, 0x6d, 0x65, 0x74, 0x79,
		0x70, 0x65, 0x22, 0x3a, 0x2d, 0x33, 0x37, 0x39,
		0x31, 0x38, 0x34, 0x39, 0x36, 0x31, 0x30, 0x37,
		0x34, 0x30, 0x34, 0x35, 0x33, 0x35, 0x31, 0x37,
		0x2c, 0x22, 0x61, 0x70, 0x70, 0x69, 0x64, 0x22,
		0x3a, 0x22, 0x31, 0x33, 0x36, 0x39, 0x30, 0x37,
		0x38, 0x34, 0x30, 0x39, 0x38, 0x37, 0x33, 0x34,
		0x30, 0x32, 0x22, 0x7d, 0x00, 0x07, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x41, 0x87, 0x51,
		0xfd, 0x39, 0x7e, 0x1c, 0x77, 0x02, 0x00,
	}

	packet := make([]Message, 0)
	err := Unmarshal(data, &packet)
	if err != nil {
		t.Error(err)
	}
	got, ok := packet[0].(*LobbyFindSessionRequest)
	if !ok {
		t.Error("failed to cast")
	}

	want := LobbyFindSessionRequest{
		VersionLock:  0xc62f01d78f77910d,
		Mode:         ModeArenaPublic,
		Level:        0xffffffffffffffff,
		Platform:     ToSymbol("DMO"),
		LoginSession: uuid.Must(uuid.FromString("0a223cf0-8b73-3dfa-2ee8-7774c3b28097")),
		Unk1:         769,
		SessionSettings: SessionSettings{
			AppId: "1369078409873402",
			Mode:  -3791849610740453517,
			Level: nil,
		},
		EvrId:     *lo.Must(ParseEvrId("DMO-8582873777389537089")),
		TeamIndex: 2,
	}

	if *got != want {
		t.Error(got, want)
	}

}

func TestLobbyFindSessionRequest_Unpack(t *testing.T) {
	data := []byte{
		0xf6, 0x40, 0xbb, 0x78, 0xa2, 0xe7, 0x8c, 0xbb,
		0xf5, 0xa3, 0x9a, 0x81, 0x01, 0x2a, 0x2c, 0x31,
		0xa7, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x0d, 0x91, 0x77, 0x8f, 0xd7, 0x01, 0x2f, 0xc6,
		0x73, 0xaf, 0x1c, 0x7e, 0xde, 0xa4, 0x60, 0xcb,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xe5, 0xef, 0x94, 0xa8, 0xb1, 0xd0, 0xe8, 0xc8,
		0xf0, 0x3c, 0x22, 0x0a, 0x73, 0x8b, 0xfa, 0x3d,
		0x2e, 0xe8, 0x77, 0x74, 0xc3, 0xb2, 0x80, 0x97,
		0x01, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x7b, 0x22, 0x67, 0x61, 0x6d, 0x65, 0x74, 0x79,
		0x70, 0x65, 0x22, 0x3a, 0x2d, 0x33, 0x37, 0x39,
		0x31, 0x38, 0x34, 0x39, 0x36, 0x31, 0x30, 0x37,
		0x34, 0x30, 0x34, 0x35, 0x33, 0x35, 0x31, 0x37,
		0x2c, 0x22, 0x61, 0x70, 0x70, 0x69, 0x64, 0x22,
		0x3a, 0x22, 0x31, 0x33, 0x36, 0x39, 0x30, 0x37,
		0x38, 0x34, 0x30, 0x39, 0x38, 0x37, 0x33, 0x34,
		0x30, 0x32, 0x22, 0x7d, 0x00, 0x07, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x41, 0x87, 0x51,
		0xfd, 0x39, 0x7e, 0x1c, 0x77, 0x02, 0x00,
	}

	packet := make([]Message, 0)
	err := Unmarshal(data, &packet)
	if err != nil {
		t.Error(err)
	}
	got, ok := packet[0].(*LobbyFindSessionRequest)
	if !ok {
		t.Error("failed to cast")
	}

	want := LobbyFindSessionRequest{
		VersionLock:  0xc62f01d78f77910d,
		Mode:         ModeArenaPublic,
		Level:        0xffffffffffffffff,
		Platform:     ToSymbol("DMO"),
		LoginSession: uuid.Must(uuid.FromString("0a223cf0-8b73-3dfa-2ee8-7774c3b28097")),
		Unk1:         769,
		SessionSettings: SessionSettings{
			AppId: "1369078409873402",
			Mode:  -3791849610740453517,
			Level: nil,
		},
		EvrId:     *lo.Must(ParseEvrId("DMO-8582873777389537089")),
		TeamIndex: 2,
	}

	if *got != want {
		t.Error(got, want)
	}

}
