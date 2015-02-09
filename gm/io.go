// Copyright 2012 Dorival de Moraes Pedroso. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gm

import (
	"bytes"
	"encoding/json"

	"code.google.com/p/gosl/utl"
)

const (
	NURBS_GEO = 18 // geometry type in geodefs
)

// HashPoint returns a unique id of a point
func HashPoint(x, y, z float64) int {
	return int(x*10001 + y*1000001 + z*100000001)
}

// WriteMshD writes .msh file
// Input: vtagged maps hashed id of control point to vertex tag
//        ctagged maps idOfNurbs_localIdOfElem to cell tag
func WriteMshD(dirout, fnk string, nurbss []*Nurbs, vtagged map[int]int, ctagged map[string]int) {
	var buf bytes.Buffer
	utl.Ff(&buf, "{\n  \"verts\" : [\n")
	verts := make(map[int]int)
	vid := 0
	for _, o := range nurbss {
		for k := 0; k < o.n[2]; k++ {
			for j := 0; j < o.n[1]; j++ {
				for i := 0; i < o.n[0]; i++ {
					x := o.GetQ(i, j, k)
					hsh := HashPoint(x[0], x[1], x[2])
					if _, ok := verts[hsh]; !ok {
						tag := 0
						if vtagged != nil {
							if val, tok := vtagged[hsh]; tok {
								tag = val
							}
						}
						if len(verts) > 0 {
							utl.Ff(&buf, ",\n")
						}
						utl.Ff(&buf, "    { \"id\":%3d, \"tag\":%3d, \"c\":[%24.17e,%24.17e,%24.17e,%24.17e] }", vid, tag, x[0], x[1], x[2], x[3])
						verts[hsh] = vid
						vid += 1
					}
				}
			}
		}
	}
	utl.Ff(&buf, "\n  ],\n  \"nurbss\" : [\n")
	for sid, o := range nurbss {
		if sid > 0 {
			utl.Ff(&buf, ",\n")
		}
		utl.Ff(&buf, "    { \"id\":%d, \"gnd\":%d, \"ords\":[%d,%d,%d],\n", sid, o.gnd, o.p[0], o.p[1], o.p[2])
		utl.Ff(&buf, "      \"knots\":[\n")
		for d := 0; d < o.gnd; d++ {
			if d > 0 {
				utl.Ff(&buf, ",\n")
			}
			utl.Ff(&buf, "        [")
			for i, t := range o.b[d].T {
				if i > 0 {
					utl.Ff(&buf, ",")
				}
				utl.Ff(&buf, "%24.17e", t)
			}
			utl.Ff(&buf, "]")
		}
		utl.Ff(&buf, "\n      ],\n      \"ctrls\":[")
		first := true
		for k := 0; k < o.n[2]; k++ {
			for j := 0; j < o.n[1]; j++ {
				for i := 0; i < o.n[0]; i++ {
					if !first {
						utl.Ff(&buf, ",")
					}
					x := o.GetQ(i, j, k)
					hsh := HashPoint(x[0], x[1], x[2])
					utl.Ff(&buf, "%d", verts[hsh])
					if first {
						first = false
					}
				}
			}
		}
		utl.Ff(&buf, "]\n    }")
	}
	utl.Ff(&buf, "\n  ],\n  \"cells\" : [\n")
	cid := 0
	for sid, o := range nurbss {
		elems := o.Elements()
		enodes := o.Enodes()
		for eid, e := range elems {
			if cid > 0 {
				utl.Ff(&buf, ",\n")
			}
			tag := -1
			if ctagged != nil {
				if val, tok := ctagged[utl.Sf("%d_%d", sid, eid)]; tok {
					tag = val
				}
			}
			utl.Ff(&buf, "    { \"id\":%3d, \"tag\":%2d, \"nrb\":%d, \"part\":0, \"geo\":%d,", cid, tag, sid, NURBS_GEO)
			utl.Ff(&buf, " \"span\":[")
			for k, idx := range e {
				if k > 0 {
					utl.Ff(&buf, ",")
				}
				utl.Ff(&buf, "%d", idx)
			}
			utl.Ff(&buf, "], \"verts\":[")
			for i, l := range enodes[eid] {
				if i > 0 {
					utl.Ff(&buf, ",")
				}
				x := o.GetQl(l)
				hsh := HashPoint(x[0], x[1], x[2])
				utl.Ff(&buf, "%d", verts[hsh])
			}
			utl.Ff(&buf, "] }")
			cid += 1
		}
	}
	utl.Ff(&buf, "\n  ]\n}")
	utl.WriteFileVD(dirout, fnk+".msh", &buf)
}

// Vert holds data for a vertex => control point
type Vert struct {
	Id  int       // id
	Tag int       // tag
	C   []float64 // coordinates (size==4)
}

// NurbsD holds all data required to read/save a Nurbs to/from files
type NurbsD struct {
	Id    int         // id of Nurbs
	Gnd   int         // 1: curve, 2:surface, 3:volume (geometry dimension)
	Ords  []int       // order along each x-y-z direction [gnd]
	Knots [][]float64 // knots along each x-y-z direction [gnd][m]
	Ctrls []int       // global ids of control points
}

// Data holds all geometry data
type Data struct {
	Verts  []Vert   // vertices
	Nurbss []NurbsD // NURBSs
}

// ReadMsh reads .msh file
func ReadMsh(fnk string) (nurbss []*Nurbs) {

	// read file
	fn := fnk + ".msh"
	buf, err := utl.ReadFile(fn)
	if err != nil {
		utl.Panic(_io_err1, fn, err)
	}

	// decode
	var dat Data
	err = json.Unmarshal(buf, &dat)
	if err != nil {
		utl.Panic(_io_err2, fn, err)
	}

	// list of vertices
	verts := make([][]float64, len(dat.Verts))
	for i, _ := range dat.Verts {
		verts[i] = make([]float64, 4)
		for j := 0; j < 4; j++ {
			verts[i][j] = dat.Verts[i].C[j]
		}
	}

	// allocate NURBSs
	nurbss = make([]*Nurbs, len(dat.Nurbss))
	for i, v := range dat.Nurbss {
		nurbss[i] = new(Nurbs)
		nurbss[i].Init(v.Gnd, v.Ords, v.Knots)
		nurbss[i].SetControl(verts, v.Ctrls)
	}
	return
}

// error messages
var (
	_io_err1 = "ReadMsh cannot read file = '%s', %v'"
	_io_err2 = "ReadMsh cannot unmarshal file = '%s', %v'"
)