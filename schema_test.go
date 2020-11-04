package goparquet

import (
	"bytes"
	"io"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fraugster/parquet-go/parquet"
	"github.com/fraugster/parquet-go/parquetschema"
)

var sizeFixture = []struct {
	Col      *ColumnStore
	Generate func(n int) ([]interface{}, int64)
}{
	{
		Col: func() *ColumnStore {
			n, err := NewInt32Store(parquet.Encoding_PLAIN, true, &ColumnParameters{})
			if err != nil {
				panic(err)
			}
			return n
		}(),
		Generate: func(n int) ([]interface{}, int64) {
			ret := make([]interface{}, 0, n)
			var size int64
			for i := 0; i < n; i++ {
				ret = append(ret, rand.Int31())
				size += 4
			}

			return ret, size
		},
	},

	{
		Col: func() *ColumnStore {
			n, err := NewInt64Store(parquet.Encoding_PLAIN, true, &ColumnParameters{})
			if err != nil {
				panic(err)
			}
			return n
		}(),
		Generate: func(n int) ([]interface{}, int64) {
			ret := make([]interface{}, 0, n)
			var size int64
			for i := 0; i < n; i++ {
				ret = append(ret, rand.Int63())
				size += 8
			}

			return ret, size
		},
	},

	{
		Col: func() *ColumnStore {
			n, err := NewByteArrayStore(parquet.Encoding_PLAIN, true, &ColumnParameters{})
			if err != nil {
				panic(err)
			}
			return n
		}(),
		Generate: func(n int) ([]interface{}, int64) {
			ret := make([]interface{}, 0, n)
			var size int64
			for i := 0; i < n; i++ {
				s := rand.Int63n(32)
				data := make([]byte, s)
				ret = append(ret, data)
				size += s
			}

			return ret, size
		},
	},
}

func TestColumnSize(t *testing.T) {
	for _, sf := range sizeFixture {
		arr, size := sf.Generate(rand.Intn(1000) + 1)
		sf.Col.reset(parquet.FieldRepetitionType_REQUIRED, 0, 0)
		for i := range arr {
			err := sf.Col.add(arr[i], 0, 0, 0)
			require.NoError(t, err)
		}
		require.Equal(t, size, sf.Col.values.size)
	}
}

func TestSchemaCopy(t *testing.T) {
	schema := `message txn {
  optional boolean is_fraud;
}`
	def, err := parquetschema.ParseSchemaDefinition(schema)
	require.NoError(t, err)
	buf := &bytes.Buffer{}
	writer := NewFileWriter(buf, WithSchemaDefinition(def))

	for i := 0; i < 3; i++ {
		var d interface{}
		switch {
		case i%3 == 0:
			d = true
		case i%3 == 1:
			d = false
		case i%3 == 2:
			d = nil
		}
		require.NoError(t, writer.AddData(map[string]interface{}{
			"is_fraud": d,
		}))
	}

	require.NoError(t, writer.Close())

	buf2 := bytes.NewReader(buf.Bytes())
	buf3 := &bytes.Buffer{}
	reader, err := NewFileReader(buf2)
	require.NoError(t, err)
	writer2 := NewFileWriter(buf3, WithSchemaDefinition(reader.GetSchemaDefinition()))

	for {
		rec, err := reader.NextRow()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		err = writer2.AddData(rec)

		require.NoError(t, err)
	}

	require.NoError(t, writer2.Close())
}

func TestFuzzCrashReadGroupSchema(t *testing.T) {
	data := []byte("PAR1\x15\x02\x19\xfc)H!org.apach" +
		"e.impala.ComplexType" +
		"sTbl\x15\f\x00\x15\x04%\x02\x18\x02id\x004\x02\x18\t" +
		"int_array\x15\x02\x15\x06\x005\x04\x18\x04li" +
		"st\x15\x02\x00\x15\x02%\x02\x18\aelement\x005" +
		"\x02\x18\x0fint_array_Array\x15\x02" +
		"\x15\x06\x005\x04\x18\x04list\x15\x02\x005\x02\x18\ael" +
		"ement\x15\x02\x15\x06\x005\x04\x18\x04list\x15\x02" +
		"\x00\x15\x02%\x02\x18\aelement\x005\x02\x18\ai" +
		"nt_map\x15\x02\x15\x02\x005\x04\x18\x03map\x15\x04" +
		"\x15\x04\x00\x15\f%\x00\x18\x03key%\x00\x00\x15\x02%\x02\x18" +
		"\x05value\x005\x02\x18\rint_Map_A" +
		"rray\x15\x02\x15\x06\x005\x04\x18\x04list\x15\x02\x00" +
		"5\x02\x18\aelement\x15\x02\x15\x02\x005\x04\x18\x03" +
		"map\x15\x04\x15\x04\x00\x15\f%\x00\x18\x03key%\x00\x00" +
		"\x15\x02%\x02\x18\x05value\x005\x02\x18\rnest" +
		"ed_struct\x15\b\x00\x15\x02%\x02\x18\x01A\x00" +
		"5\x02\x18\x01b\x15\x02\x15\x06\x005\x04\x18\x04list\x15\x02" +
		"\x00\x15\x02%\x02\x18\aelement\x005\x02\x18\x01C" +
		"\x15\x02\x005\x02\x18\x01d\x15\x02\x15\x06\x005\x04\x18\x04lis" +
		"t\x15\x02\x005\x02\x18\aelement\x15\x02\x15\x06\x00" +
		"5\x04\x18\x04list\x15\x02\x005\x02\x18\aeleme" +
		"nt\x15\x04\x00\x15\x02%\x02\x18\x01E\x00\x15\f%\x02\x18\x01F" +
		"%\x00\x005\x02\x18\x01g\x15\x02\x15\x02\x005\x04\x18\x03map" +
		"\x15\x04\x15\x04\x00\x15\f%\x00\x18\x03key%\x00\x005\x02\x18" +
		"\x05value\x15\x02\x005\x02\x18\x01H\x15\x02\x005\x02\x18" +
		"\x01i\x15\x02\x15\x06\x005\x04\x18\x04list\x15\x02\x00\x15\n" +
		"%\x02\x18\aelement\x00\x16\x0e\x19\x1c\x19\xdc&\b" +
		"\x1c\x15\x04\x195\b\x00\x06\x19\x18\x02id\x15\x00\x16\x0e\x16\xce\x01" +
		"\x16\xce\x01&\b<\x18\b\a\x00\x00\x00\x00\x00\x00\x00\x18\b\x01\x00" +
		"\x00\x00\x00\x00\x00\x00\x16\x00\x00\x00\x00&\xd6\x01\x1c\x15\x02\x19%\x06" +
		"\x04\x198\tint_array\x04list\ae" +
		"lement\x15\x00\x16\x1c\x16\x9c\x01\x16\x9c\x01&\xd6\x01<" +
		"\x18\x04\x03\x00\x00\x00\x18\x04\x01\x00\x00\x00\x16\x10\x00\x00\x00&\xf2\x02" +
		"\x1c\x15\x02\x19%\x06\x04\x19X\x0fint_array_" +
		"Array\x04list\aelement\x04l" +
		"ist\aelement\x15\x00\x16(\x16\xce\x01\x16\xce" +
		"\x01&\xf2\x02<\x18\x04\x06\x00\x00\x00\x18\x04\x01\x00\x00\x00\x16\x14\x00" +
		"\x00\x00&\xc0\x04\x1c\x15\f\x19%\x06\x04\x198\aint_m" +
		"ap\x03map\x03key\x15\x00\x16\x14\x16\xa0\x01\x16\xa0\x01" +
		"&\xc0\x04<\x18\x02k3\x18\x02k1\x16\b\x00\x00\x00&\xe0\x05" +
		"\x1c\x15\x02\x19%\x00\x06\x198\aint_map\x03ma" +
		"p\x05value\x15\x00\x16\x14\x16z\x16z&\xe0\x05<\x18" +
		"\x04d\x00\x00\x00\x18\x04\x01\x00\x00\x00\x16\x0e\x00\x00\x00&\xda\x06\x1c" +
		"\x15\f\x19%\x06\x04\x19X\rint_Map_Arr" +
		"ay\x04list\aelement\x03map\x03" +
		"key\x15\x00\x16\x16\x16\x9a\x01\x16\x9a\x01&\xda\x06<\x18\x02k" +
		"3\x18\x02k1\x16\x10\x00\x00\x00&\xf4\a\x1c\x15\x02\x19%\x06\x04" +
		"\x19X\rint_Map_Array\x04lis" +
		"t\aele-ent\x03map\x05value\x15" +
		"\x00\x16\x16\x16\x90\x01\x16\x90\x01&\xf4\a<\x18\x04\x01\x00\x00\x00\x18" +
		"\x04\x01\x00\x00\x00\x16\x12\x00\x00\x00&\x84\t\x1c\x15\x02\x195\b\x00" +
		"\x06\x19(\rnested_struct\x01A\x15" +
		"\x00\x16\x0e\x16`\x16`&\x84\t<\x18\x04\a\x00\x00\x00\x18\x04\x01" +
		"\x00\x00\x00\x16\n\x00\x00\x00&\xe4\t\x1c\x15\x02\x19%\x00\x06\x19H" +
		"\rnested_struct\x01b\x04lis" +
		"t\aelement\x15\x00\x16\x12\x16~\x16~&\xe4\t" +
		"<\x18\x04\x03\x00\x00\x00\x18\x04\x01\x00\x00\x00\x16\f\x00\x00\x00&\xe2" +
		"\n\x1c\x15\x02\x19%\x06\x04\x19\x88\rnested_st" +
		"ruct\x01C\x01d\x04list\aelemen" +
		"t\x04list\ael\xecment\x01E\x15\x00\x16&" +
		"\x16\xb4\x01\x16\xb4\x01&\xe2\n<\x18\x04\v\x00\x00\x00\x18\x04\xf6\xff" +
		"\xff\xff\x16\x1a\x00\x00\x00&\x96\f\x1c\x15\f\x19%\x06\x04\x19\x88\r" +
		"nested_struct\x01C\x01d\x04li" +
		"st\aelement\x04list\aelem" +
		"ent\x01F\x15\x00\x16&\x16\xba\x01\x16\xba\x01&\x96\f<\x18" +
		"\x01c\x18\x03aaa\x16\x1a\x00\x00\x00&\xd0\r\x1c\x15\f\x19%" +
		"\x06\x04\x19H\rnested_struct\x01g" +
		"\x03map\x03key\x15\x00\x16\x16\x16\xca\x01\x16\xca\x01&\xd0" +
		"\r<\x18\x02g5\x18\x03foo\x16\b\x00\x00\x00&\x9a\x0f\x1c" +
		"\x15\n\x19%\x06\x04\x19\x88\rnested_stru" +
		"ct\x01g\x03map\x05value\x01H\x01i\x04l" +
		"ist\aelement\x15\x00\x16\x1a\x16\xd0\x01\x16\xd0" +
		"\x01&\x9a\x0f<\x18\bffffff\n@\x18\b\x9a\x99\x99" +
		"\x99\x99\x99\xf1?\x16\x12\x00\x00\x00\x16\xe2\x10\x16\x0e\x00\x19\x1c\x18\x13" +
		"pa:quet.avro.schema\x18" +
		"\xc0\t{\"type\":\"record\",\"" +
		"name\":\"ComplexTypesT" +
		"bl\",\"namespace\":\"org" +
		".apache.impala\",\"fie" +
		"lds\":[{\"name\":\"id\",\"" +
		"type\":[\"null\",\"long\"" +
		"]},{\"name\":\"int_arra" +
		"y\",\"type\":[\"null\",{\"" +
		"type\":\"array\",\"ioems" +
		"\":[\"null\",\"int\"]}]}," +
		"{\"name\":\"int_array_A" +
		"rray\",\"type\":[\"null\"" +
		",{\"type\":\"array\",\"it" +
		"ems\":[\"null\",{\"type\"" +
		":\"array\",\"items\":[\"n" +
		"ull\",\"int\"]}]}]},{\"n" +
		"ame\":\"int_map\",\"type" +
		"\":[\"null\",{\"type\":\"m" +
		"ap\",\"values\":[\"null\"" +
		",\"int\"]}]},{\"na\xfbe\":\"" +
		"int_Map_Array\",\"type" +
		"\":[\"null\",{\"type\":\"a" +
		"rray\",\"items\":[\"null" +
		"\",{\"type\":\"map\",\"val" +
		"ues\":[\"null\",\"int\"]}" +
		"]}]},{\"name\":\"nested" +
		"_struct\",\"type\":[\"nu" +
		"ll\",{\"type\":\"record\"" +
		",\"name\":\"r1\",\"fields" +
		"\":[{\"name\":\"A\",\"type" +
		"\":[ null\",\"int\"]},{\"" +
		"name\":\"b\",\"type\":[\"n" +
		"ull\",{\"type\":\"array\"" +
		",\"items\":[\"nulc\",\"in" +
		"t\"]}]},{\"name\":\"C\",\"" +
		"type\":[\"null\",{\"type" +
		"\":\"record\",\"name\":\"r" +
		"2\",\"fields\":[{\"name\"" +
		":\"d\",\"type\":[\"null\"," +
		"{\"type\":\"array\",\"ite" +
		"ms\":[\"null\",{\"type\":" +
		"\"array\",\"items\":[\"nu" +
		"ll\",{\"type\":\"record\"" +
		",\"name\":\"r3\",\"fields" +
		"\":[{\"name\":\"E\",\"type" +
		"\":[\"null\",\"int\"]},{\"" +
		"name\":\"F\",\"type\":[\"n" +
		"ull\",\"string\"]}]}]}]" +
		"}]}]}]},{\"name\":\"g\"," +
		"\"type\":[\"null\",{\"typ" +
		"e\":\"map\",\"values\":[\"" +
		"null\",{\"type\":\"recor" +
		"d\",\"name\":\"r4\",\"fiel" +
		"ds\":[{\"name\":\"H\",\"ty" +
		"pe\":[\"null\",{\"type\":" +
		"\"record\",\"name\":\"r5\"" +
		",\"fields\":[{\"name\":\"" +
		"i\",\"type\":[\"nul\xef\xff\xff\xff\xff" +
		"\xff\xff\xffe\":\"array\",\"items" +
		"\":[\"null\",\"double\"]}" +
		"]}]}]}]}]}]}]}]}]}\x00\x18" +
		"Iparquet-mr version " +
		"1.8.0 (bui,d 0fda28a" +
		"f84b9746396014ad6a41" +
		"5b90592a98b3b)\x00\xfb\n\x00\x00P" +
		"AR1")

	readAllData(t, data)
}

func readAllData(t *testing.T, data []byte) {
	r, err := NewFileReader(bytes.NewReader(data))
	if err != nil {
		t.Logf("NewFileReader returned error: %v", err)
		return
	}

	rows := r.NumRows()
	for i := int64(0); i < rows; i++ {
		_, err := r.NextRow()
		if err != nil {
			t.Logf("NextRow returned error: %v", err)
			return
		}
	}
}
