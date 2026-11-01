package fileformat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileFormatConstantsAndHelpers(t *testing.T) {
	assert.Equal(t, []string{".si4", ".sg4", ".sn4"}, ScidV4Extensions())
	assert.Equal(t, []string{".si4", ".sg4", ".sn4", ".pgn"}, AllDatabaseExtensions())

	assert.Equal(t, "/data/mygames", TrimScidExtensions("/data/mygames.si4"))
	assert.Equal(t, "/data/mygames", TrimScidExtensions("/data/mygames.sg4"))
	assert.Equal(t, "/data/mygames", TrimScidExtensions("/data/mygames.sn4"))
	assert.Equal(t, "/data/mygames", TrimScidExtensions("/data/mygames.si3"))
	assert.Equal(t, "/data/mygames", TrimDatabaseExtensions("/data/mygames.PGN"))
	assert.Equal(t, "/data/mygames", TrimDatabaseExtensions("/data/mygames.si4"))

	assert.Equal(t, "/data/mygames.si4", ScidIndexPath("/data/mygames.sg4"))
	assert.Equal(t, "/data/mygames.sg4", ScidGameDataPath("/data/mygames.si4"))
	assert.Equal(t, "/data/mygames.sn4", ScidNamesPath("/data/mygames"))

	si, sn, sg := ScidPaths("/data/club.si4")
	assert.Equal(t, "/data/club.si4", si)
	assert.Equal(t, "/data/club.sn4", sn)
	assert.Equal(t, "/data/club.sg4", sg)

	assert.True(t, IsPGNPath("games.PGN"))
	assert.False(t, IsPGNPath("games.si4"))
	assert.True(t, IsScidIndexPath("club.SI4"))
	assert.False(t, IsScidIndexPath("club.sg4"))

	assert.True(t, IsScidComponentExt(".si4"))
	assert.True(t, IsScidComponentExt(".SG4"))
	assert.True(t, IsScidComponentExt(".sn4"))
	assert.False(t, IsScidComponentExt(".pgn"))

	assert.True(t, IsPolyglotBookPath("Performance.BIN"))
	assert.True(t, IsJSONPath("preferences.JSON"))
	assert.True(t, HasExtension("archive.7z", Ext7z))
}
