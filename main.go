package main 

import (
	"net" 
	"log" 
	"sync" 
	"strings" 
) 

// server dtype 
type Server struct {
	Addr 		string 	
	Ln		net.Listener 		
} 

func NewServer(addr string) *Server {
	return &Server {
		Addr : addr,  
	}  
} 

func (s *Server) Listen() error {
	ln, err := net.Listen("tcp", s.Addr); 
	s.Ln = ln; 
	return err; 
} 


// user dtype
type User struct {
	Addr 		net.Addr 
	RoomId 		RoomID
	conn 		*net.Conn 
	Room 		*Room
} 

func (u *User) GetConnection() *net.Conn {
	return u.conn;
}  

func (u *User) GetARoomYouGuys() *Room {
	return u.Room; 
} 


// room dtype 
type RoomID int 

type Room struct {
	RoomId 		RoomID 
	users 		[] User
	RoomName 	string  
} 


func (r *Room) GetUsers() [] User {
	return r.users; 
} 

func (r * Room) AddUser(user User) bool {
	// returns false if user already exists in the room to indicate failure in addition  
	users := r.GetUsers();  
	for _, roomUser := range users {
		if roomUser.Addr.String() == user.Addr.String() {
			return false  
		}  
	} 

	r.users = append(r.users, user);  
	return true; 
} 


type ErrorChan struct {
	mu 		sync.Mutex 
	ErrMap 		map[string] error 
} 

func (ec *ErrorChan) AddNewRoutineError(routineAddr string, err error) {
	// error in chastity 
	ec.mu.Lock(); 
	defer ec.mu.Unlock(); 
	ec.ErrMap[routineAddr] = err; 
}  

// distribute message from user to other users of the group 
func (r *Room) DistributeMsg(fromUser string, msg string) *ErrorChan {
	errChan := ErrorChan{
		ErrMap: make(map[string] error), 
	}   

	// wait group initialized 
	var wg sync.WaitGroup; 
	for _, usr := range r.GetUsers() {
		if usr.Addr.String() != fromUser {

			// send a routine to write messages to everyone at the same freakin time bro 
			// I would marry all go-routines someday 
		
			// Adding a unique routine to the wait group 
			wg.Add(1); 
			go func () {
				conn := *usr.GetConnection(); 
				addstr := strings.Split(fromUser, ":"); 
				
				_, err := conn.Write([] byte(
					"/" + addstr[len(addstr) - 1] + ": " + msg + "\n",   	
				)); 
				if err != nil {
					errChan.AddNewRoutineError(
						conn.RemoteAddr().String(), 
						err, 
					); 
				} 	

				// routine sends done message to the wait group 
				wg.Done() 
			}()  
			
		}  
	} 

	// wait group waits for all the routines to be finished 
	wg.Wait(); 
	return &errChan; 
} 



func main() {
	server := NewServer(":2000"); 
	err := server.Listen(); 

	var roomId RoomID = 10; 
	room := Room {
		RoomId: 	roomId, 
		RoomName: 	"common-room", 	
	} 

	if err != nil {
		log.Fatalln(
			"Could not listen to port because skill issues: %s\n",
			 err.Error()); 
	} 
	
	for {
		// accept incoming connection 
		conn, err := server.Ln.Accept(); 
		if err != nil {
			log.Printf(
				"Attempt to connect with server failed by Addr: %s\n", 
				conn.RemoteAddr()); 
		} 
		
		// create user 
		usr := User{
			Addr: conn.RemoteAddr(), 
			conn: &conn, 
			RoomId: roomId,  
			Room: &room, 
		} 
		
		// if user doesn't exist 
		room.AddUser(usr); 

		// spin up go-routines to read and write from user to server 
		go HandleIncoming(usr); 
	} 
} 

// a go-routine connection between server and user. 
func HandleIncoming(user User) {

	// this function will read from a specific user and then send the message to all the other users
	// that belong to the same room as the user sending the message  
	conn := *user.GetConnection();  
	for {
		// read the msg sent from the user  
		buf := make([] byte, 1024);  
		n, err := conn.Read(buf); 	
		if err != nil {
			log.Println("Could not read message from user: %s\n", err.Error());  
		} 
		
		// msg read successfully. Now we must attack the other users. 	
		msg := string(buf[:n]); 
		room := *user.GetARoomYouGuys();  
		
		// send msg to all the users in the room
		errChan := room.DistributeMsg(conn.RemoteAddr().String(), msg); 
		if len(errChan.ErrMap) != 0 {
			log.Println("Some users didn't recieve the message"); 
		} 
		
	} 
} 
