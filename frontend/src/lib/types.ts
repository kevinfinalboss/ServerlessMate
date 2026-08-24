export type Result = 'win' | 'loss' | 'draw'

export interface Players {
  white: string
  black: string
}

export interface GameState {
  gameId: string
  fen: string
  pgn: string
  players: Players
  turnOf: string
  status: string
  whiteTimeMs: number
  blackTimeMs: number
  lastMoveAt: number
  endedAt?: number
  winner?: string
  drawOfferedBy?: string
  vsAI: boolean
  aiLevel?: string
  comment?: string
}

export interface ConnectedMessage {
  type: 'connected'
  playerId: string
  isGuest: boolean
  gameId: string
  role: string
}

export interface ChatMessage {
  type: 'chat'
  playerId: string
  message: string
}

export interface ErrorMessage {
  type: 'error'
  message: string
}

export interface QueueJoinedMessage {
  type: 'queueJoined'
  matchmakingKey: string
}

export interface MatchFoundMessage {
  type: 'matchFound'
  gameId: string
}

export interface QueueLeftMessage {
  type: 'queueLeft'
}

export interface ProfileResponse {
  type?: never
  playerId: string
  username: string
  visible: boolean
  rating?: number
  wins?: number
  losses?: number
  draws?: number
  gamesPlayed?: number
}

export interface LeaderboardEntry {
  playerId: string
  username: string
  rating: number
}

export interface LeaderboardMessage {
  type: 'leaderboard'
  entries: LeaderboardEntry[]
}

export interface HistoryEntry {
  playerId: string
  endedAt: number
  gameId: string
  opponentId: string
  result: Result
  vsAI: boolean
}

export interface HistoryMessage {
  type: 'history'
  entries: HistoryEntry[]
}

export interface ReplayMessage {
  type: 'replay'
  gameId: string
  pgn: string
}

export interface FriendEventMessage {
  type: 'friendRequestSent' | 'friendRequestAccepted' | 'playerBlocked' | 'friendRequestCancelled'
  friendId: string
}

export interface FriendEntry {
  playerId: string
  username: string
}

export interface FriendsMessage {
  type: 'friends'
  friends: FriendEntry[]
  incomingRequests: FriendEntry[]
  outgoingRequests: FriendEntry[]
}

export type ServerMessage =
  | GameState
  | ConnectedMessage
  | ChatMessage
  | ErrorMessage
  | QueueJoinedMessage
  | MatchFoundMessage
  | QueueLeftMessage
  | ProfileResponse
  | LeaderboardMessage
  | HistoryMessage
  | ReplayMessage
  | FriendEventMessage
  | FriendsMessage

export function isGameState(msg: ServerMessage): msg is GameState {
  return 'fen' in msg && 'gameId' in msg
}

export function isProfileResponse(msg: ServerMessage): msg is ProfileResponse {
  return 'username' in msg && 'visible' in msg && !('fen' in msg)
}

type TaggedMessage = Extract<ServerMessage, { type: string }>

export function isType<T extends TaggedMessage>(msg: ServerMessage, type: T['type']): msg is T {
  return 'type' in msg && msg.type === type
}
