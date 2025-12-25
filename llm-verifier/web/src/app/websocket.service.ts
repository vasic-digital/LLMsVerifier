import { Injectable } from '@angular/core';
import { Subject, Observable } from 'rxjs';

export interface WebSocketMessage {
  type: 'subscribe' | 'unsubscribe' | 'event' | 'verification_update' | 'system_health' | 'error' | 'heartbeat';
  data: any;
  timestamp: string;
  id?: string;
}

export interface RealtimeEvent {
  id: string;
  type: 'model.verified' | 'model.verification.failed' | 'verification.started' | 'verification.completed' | 'system.health.changed';
  severity: 'info' | 'warning' | 'error' | 'critical';
  message: string;
  data?: any;
  timestamp: string;
  modelId?: string;
  provider?: string;
  score?: number;
}

@Injectable({
  providedIn: 'root'
})
export class WebSocketService {
  private socket: WebSocket | null = null;
  private messageSubject = new Subject<WebSocketMessage>();
  private eventSubject = new Subject<RealtimeEvent>();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private isConnecting = false;
  private heartbeatInterval: any;
  private heartbeatDelay = 30000; // 30 seconds

  public messages$: Observable<WebSocketMessage>;
  public events$: Observable<RealtimeEvent>;
  public connected$ = new Subject<boolean>();

  constructor() {
    this.messages$ = this.messageSubject.asObservable();
    this.events$ = this.eventSubject.asObservable();
  }

  connect(url: string): void {
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      return;
    }

    if (this.isConnecting) {
      return;
    }

    this.isConnecting = true;

    try {
      this.socket = new WebSocket(url);
      this.setupEventHandlers();
    } catch (error) {
      console.error('Failed to create WebSocket connection:', error);
      this.isConnecting = false;
      this.scheduleReconnect();
    }
  }

  disconnect(): void {
    if (this.socket) {
      this.socket.close();
      this.socket = null;
    }
    this.reconnectAttempts = 0;
    this.isConnecting = false;
    this.connected$.next(false);
    this.stopHeartbeat();
  }

  send(message: WebSocketMessage): void {
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(message));
    } else {
      console.warn('WebSocket not connected, message not sent:', message);
    }
  }

  sendEvent(event: RealtimeEvent): void {
    const message: WebSocketMessage = {
      type: 'event',
      data: event,
      timestamp: new Date().toISOString()
    };
    this.send(message);
  }

  subscribe(events: string[]): void {
    const message: WebSocketMessage = {
      type: 'subscribe',
      data: { events },
      timestamp: new Date().toISOString()
    };
    this.send(message);
  }

  unsubscribe(events: string[]): void {
    const message: WebSocketMessage = {
      type: 'unsubscribe',
      data: { events },
      timestamp: new Date().toISOString()
    };
    this.send(message);
  }

  isConnected(): boolean {
    return this.socket !== null && this.socket.readyState === WebSocket.OPEN;
  }

  private setupEventHandlers(): void {
    if (!this.socket) {
      return;
    }

    this.socket.onopen = () => {
      console.log('WebSocket connected');
      this.isConnecting = false;
      this.reconnectAttempts = 0;
      this.connected$.next(true);
      this.startHeartbeat();

      // Subscribe to default events for real-time updates
      this.subscribe([
        'model.verified',
        'model.verification.failed',
        'verification.started',
        'verification.completed',
        'system.health.changed',
        'provider.status.changed',
        'brotli.test.completed'
      ]);
    };

    this.socket.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data) as WebSocketMessage;
        this.messageSubject.next(message);

        if (message.type === 'event' && message.data) {
          this.eventSubject.next(message.data as RealtimeEvent);
        }
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error);
      }
    };

    this.socket.onclose = (event) => {
      console.log('WebSocket disconnected:', event);
      this.connected$.next(false);
      this.isConnecting = false;
      this.stopHeartbeat();

      if (!event.wasClean && this.reconnectAttempts < this.maxReconnectAttempts) {
        this.scheduleReconnect();
      }
    };

    this.socket.onerror = (error) => {
      console.error('WebSocket error:', error);
      this.isConnecting = false;
      this.connected$.next(false);
    };
  }

  private startHeartbeat(): void {
    this.stopHeartbeat();
    this.heartbeatInterval = setInterval(() => {
      if (this.isConnected()) {
        const heartbeatMessage: WebSocketMessage = {
          type: 'heartbeat',
          data: { timestamp: new Date().toISOString() },
          timestamp: new Date().toISOString()
        };
        this.send(heartbeatMessage);
      }
    }, this.heartbeatDelay);
  }

  private stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      return;
    }

    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts);
    console.log(`Scheduling reconnection attempt ${this.reconnectAttempts + 1} in ${delay}ms`);

    setTimeout(() => {
      this.reconnectAttempts++;
      // Recreate connection using the same URL
      if (this.socket && this.socket.url) {
        this.connect(this.socket.url);
      }
    }, delay);
  }
}