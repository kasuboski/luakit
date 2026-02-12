package dag

import (
	pb "github.com/moby/buildkit/solver/pb"
)

// SourceMapBuilder helps build the pb.Source mapping for a DAG.
type SourceMapBuilder struct {
	sourceInfos  []*pb.SourceInfo
	fileIndexMap map[string]int
	locations    map[string]*pb.Locations
}

// NewSourceMapBuilder creates a new SourceMapBuilder.
func NewSourceMapBuilder() *SourceMapBuilder {
	return &SourceMapBuilder{
		sourceInfos:  []*pb.SourceInfo{},
		fileIndexMap: make(map[string]int),
		locations:    make(map[string]*pb.Locations),
	}
}

// AddFile adds a Lua source file to the source map and returns its index.
// If the file already exists, it returns the existing index.
func (smb *SourceMapBuilder) AddFile(filename string, data []byte) int {
	if idx, exists := smb.fileIndexMap[filename]; exists {
		return idx
	}

	idx := len(smb.sourceInfos)
	smb.sourceInfos = append(smb.sourceInfos, &pb.SourceInfo{
		Filename: filename,
		Data:     data,
		Language: "Lua",
	})
	smb.fileIndexMap[filename] = idx
	return idx
}

// AddLocation adds a location mapping for an operation to a line in a file.
func (smb *SourceMapBuilder) AddLocation(digest string, filename string, line int) {
	fileIdx, exists := smb.fileIndexMap[filename]
	if !exists {
		return
	}

	if fileIdx < 0 || fileIdx > int(^uint32(0)) || line < 0 || line > int(^uint32(0)) {
		return
	}
	if locations, ok := smb.locations[digest]; ok {
		locations.Locations = append(locations.Locations, &pb.Location{
			SourceIndex: int32(fileIdx),
			Ranges: []*pb.Range{
				{
					Start: &pb.Position{
						Line: int32(line),
					},
					End: &pb.Position{
						Line: int32(line),
					},
				},
			},
		})
	} else {
		smb.locations[digest] = &pb.Locations{
			Locations: []*pb.Location{
				{
					SourceIndex: int32(fileIdx),
					Ranges: []*pb.Range{
						{
							Start: &pb.Position{
								Line: int32(line),
							},
							End: &pb.Position{
								Line: int32(line),
							},
						},
					},
				},
			},
		}
	}
}

// Build creates the pb.Source object with all collected info.
func (smb *SourceMapBuilder) Build() *pb.Source {
	return &pb.Source{
		Infos:     smb.sourceInfos,
		Locations: smb.locations,
	}
}
