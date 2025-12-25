import { TestBed } from '@angular/core/testing';
import { WebSocketService, WebSocketMessage, RealtimeEvent } from './websocket.service';

describe('WebSocketService', () => {
  let service: WebSocketService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(WebSocketService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should connect to WebSocket', () => {
    const mockWebSocket = {
      readyState: 1,
      close: () => {},
      url: 'ws://localhost:8080/ws'
    };
    
    spyOn(window, 'WebSocket').and.returnValue(mockWebSocket as any);
    
    service.connect('ws://localhost:8080/ws');
    
    expect(window.WebSocket).toHaveBeenCalledWith('ws://localhost:8080/ws');
  });

  it('should send messages when connected', () => {
    const mockSend = jasmine.createSpy('send');
    const mockWebSocket = {
      readyState: 1,
      send: mockSend,
      close: () => {}
    };
    
    spyOn(window, 'WebSocket').and.returnValue(mockWebSocket as any);
    service.connect('ws://localhost:8080/ws');
    
    const message: WebSocketMessage = {
      type: 'subscribe',
      data: { events: ['model.verified'] },
      timestamp: new Date().toISOString()
    };
    
    service.send(message);
    
    expect(mockSend).toHaveBeenCalledWith(JSON.stringify(message));
  });

  it('should not send messages when disconnected', () => {
    const mockSend = jasmine.createSpy('send');
    
    const message: WebSocketMessage = {
      type: 'subscribe',
      data: { events: ['model.verified'] },
      timestamp: new Date().toISOString()
    };
    
    service.send(message);
    
    expect(mockSend).not.toHaveBeenCalled();
  });

  it('should handle reconnection attempts', () => {
    const mockWebSocket = {
      readyState: 3, // CLOSED
      close: () => {},
      url: 'ws://localhost:8080/ws'
    };
    
    spyOn(window, 'WebSocket').and.returnValue(mockWebSocket as any);
    spyOn(service as any, 'scheduleReconnect').and.callThrough();
    
    service.connect('ws://localhost:8080/ws');
    
    // Simulate connection closure
    if (mockWebSocket.onclose) {
      mockWebSocket.onclose({ wasClean: false } as CloseEvent);
    }
    
    expect((service as any).scheduleReconnect).toHaveBeenCalled();
  });

  it('should subscribe to events', () => {
    const mockSend = jasmine.createSpy('send');
    const mockWebSocket = {
      readyState: 1,
      send: mockSend,
      close: () => {}
    };
    
    spyOn(window, 'WebSocket').and.returnValue(mockWebSocket as any);
    service.connect('ws://localhost:8080/ws');
    
    const events = ['model.verified', 'verification.started'];
    service.subscribe(events);
    
    expect(mockSend).toHaveBeenCalledWith(jasmine.stringContaining('subscribe'));
  });

  it('should handle incoming messages', (done) => {
    const mockWebSocket = {
      readyState: 1,
      close: () => {}
    };
    
    spyOn(window, 'WebSocket').and.returnValue(mockWebSocket as any);
    service.connect('ws://localhost:8080/ws');
    
    const testMessage: WebSocketMessage = {
      type: 'model.verified',
      data: { modelId: 'test-model', score: 95 },
      timestamp: new Date().toISOString()
    };
    
    service.messages$.subscribe(message => {
      expect(message).toEqual(testMessage);
      done();
    });
    
    // Simulate incoming message
    if (mockWebSocket.onmessage) {
      mockWebSocket.onmessage({ data: JSON.stringify(testMessage) } as MessageEvent);
    }
  });

  it('should emit events from incoming messages', (done) => {
    const mockWebSocket = {
      readyState: 1,
      close: () => {}
    };
    
    spyOn(window, 'WebSocket').and.returnValue(mockWebSocket as any);
    service.connect('ws://localhost:8080/ws');
    
    const testEvent: RealtimeEvent = {
      id: 'event-123',
      type: 'model.verified',
      severity: 'info',
      message: 'Model verified successfully',
      timestamp: new Date().toISOString(),
      modelId: 'test-model',
      score: 95
    };
    
    const testMessage: WebSocketMessage = {
      type: 'event',
      data: testEvent,
      timestamp: new Date().toISOString()
    };
    
    service.events$.subscribe(event => {
      expect(event).toEqual(testEvent);
      done();
    });
    
    // Simulate incoming message
    if (mockWebSocket.onmessage) {
      mockWebSocket.onmessage({ data: JSON.stringify(testMessage) } as MessageEvent);
    }
  });

  it('should handle connection errors', () => {
    const mockWebSocket = {
      readyState: 3,
      close: () => {}
    };
    
    spyOn(window, 'WebSocket').and.returnValue(mockWebSocket as any);
    spyOn(console, 'error');
    
    service.connect('ws://invalid-url');
    
    // Simulate error
    if (mockWebSocket.onerror) {
      mockWebSocket.onerror(new Event('error'));
    }
    
    expect(console.error).toHaveBeenCalled();
  });

  it('should check connection status', () => {
    const mockWebSocket = {
      readyState: 1,
      close: () => {}
    };
    
    spyOn(window, 'WebSocket').and.returnValue(mockWebSocket as any);
    service.connect('ws://localhost:8080/ws');
    
    expect(service.isConnected()).toBeTrue();
  });

  it('should disconnect properly', () => {
    const mockClose = jasmine.createSpy('close');
    const mockWebSocket = {
      readyState: 1,
      close: mockClose
    };
    
    spyOn(window, 'WebSocket').and.returnValue(mockWebSocket as any);
    service.connect('ws://localhost:8080/ws');
    
    service.disconnect();
    
    expect(mockClose).toHaveBeenCalled();
    expect(service.isConnected()).toBeFalse();
  });

  it('should send events', () => {
    const mockSend = jasmine.createSpy('send');
    const mockWebSocket = {
      readyState: 1,
      send: mockSend,
      close: () => {}
    };
    
    spyOn(window, 'WebSocket').and.returnValue(mockWebSocket as any);
    service.connect('ws://localhost:8080/ws');
    
    const event: RealtimeEvent = {
      id: 'event-456',
      type: 'model.verified',
      severity: 'success',
      message: 'Test event',
      timestamp: new Date().toISOString()
    };
    
    service.sendEvent(event);
    
    expect(mockSend).toHaveBeenCalledWith(jasmine.stringContaining('event'));
  });
});