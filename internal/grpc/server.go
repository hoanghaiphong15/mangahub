package grpc_internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/hoanghaiphong15/mangahub/internal/tcp"
	pb "github.com/hoanghaiphong15/mangahub/pkg/proto" // Alias for your generated code
)

// Server implements the generated MangaServiceServer interface
type Server struct {
	pb.UnimplementedMangaServiceServer
	DB           *sql.DB
	TCPBroadcast chan tcp.ProgressUpdate
}

// GetManga retrieves a single manga by ID [cite: 896, 1404-1413]
func (s *Server) GetManga(ctx context.Context, req *pb.GetMangaRequest) (*pb.MangaResponse, error) {
	var m pb.Manga
	var genresStr string

	query := `SELECT id, title, author, genres, status, total_chapters, description FROM manga WHERE id = ?`
	err := s.DB.QueryRow(query, req.Id).Scan(
		&m.Id, &m.Title, &m.Author, &genresStr, &m.Status, &m.TotalChapters, &m.Description,
	)

	if err == sql.ErrNoRows {
		return &pb.MangaResponse{Error: "Manga not found"}, nil
	} else if err != nil {
		log.Printf("gRPC GetManga DB Error: %v", err)
		return &pb.MangaResponse{Error: "Database error"}, nil
	}

	// Convert SQLite JSON string back to a Go slice
	var genres []string
	json.Unmarshal([]byte(genresStr), &genres)
	m.Genres = genres

	return &pb.MangaResponse{Manga: &m}, nil
}

// SearchManga finds manga by title [cite: 896, 1414-1423]
func (s *Server) SearchManga(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	query := `SELECT id, title, author, genres, status, total_chapters, description FROM manga WHERE title LIKE ? LIMIT 20`
	rows, err := s.DB.Query(query, "%"+req.Query+"%")
	if err != nil {
		log.Printf("gRPC SearchManga DB Error: %v", err)
		return &pb.SearchResponse{Error: "Database error"}, nil
	}
	defer rows.Close()

	var results []*pb.Manga
	for rows.Next() {
		m := &pb.Manga{}
		var genresStr string
		if err := rows.Scan(&m.Id, &m.Title, &m.Author, &genresStr, &m.Status, &m.TotalChapters, &m.Description); err != nil {
			continue
		}
		var genres []string
		json.Unmarshal([]byte(genresStr), &genres)
		m.Genres = genres
		results = append(results, m)
	}

	return &pb.SearchResponse{Results: results}, nil
}

// UpdateProgress updates the database and triggers a TCP broadcast [cite: 897, 1424-1432]
func (s *Server) UpdateProgress(ctx context.Context, req *pb.ProgressRequest) (*pb.ProgressResponse, error) {
	query := `UPDATE user_progress SET current_chapter = ?, updated_at = ? WHERE user_id = ? AND manga_id = ?`
	
	result, err := s.DB.Exec(query, req.Chapter, time.Now(), req.UserId, req.MangaId)
	if err != nil {
		log.Printf("gRPC UpdateProgress DB Error: %v", err)
		return &pb.ProgressResponse{Success: false, Message: "Database error"}, nil
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return &pb.ProgressResponse{Success: false, Message: "Manga not found in user library"}, nil
	}

	// Trigger the TCP Broadcast for real-time synchronization
	updateMsg := tcp.ProgressUpdate{
		UserID:    req.UserId,
		MangaID:   req.MangaId,
		Chapter:   int(req.Chapter),
		Timestamp: time.Now().Unix(),
	}

	select {
	case s.TCPBroadcast <- updateMsg:
	default:
	}

	return &pb.ProgressResponse{Success: true, Message: "Progress updated successfully"}, nil
}