"""
WebRTC Signaling Server с Prometheus метриками
"""

from aiohttp import web
import json
import logging
from datetime import datetime
from prometheus_client import Counter, Gauge, Histogram, generate_latest, REGISTRY
from prometheus_client.exposition import CONTENT_TYPE_LATEST
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Хранилище активных пользователей
users = {}  # {username: websocket}

# Prometheus метрики
WEBSOCKET_CONNECTIONS = Gauge(
    'webrtc_websocket_connections',
    'Number of active WebSocket connections'
)

USERS_CONNECTED = Gauge(
    'webrtc_users_connected',
    'Number of connected users'
)

MESSAGES_TOTAL = Counter(
    'webrtc_messages_total',
    'Total number of WebSocket messages',
    ['message_type']
)

CALLS_TOTAL = Counter(
    'webrtc_calls_total',
    'Total number of calls initiated',
    ['call_type']
)

ACTIVE_CALLS = Gauge(
    'webrtc_active_calls',
    'Number of currently active calls'
)

CALL_DURATION = Histogram(
    'webrtc_call_duration_seconds',
    'Call duration in seconds'
)

ERRORS_TOTAL = Counter(
    'webrtc_errors_total',
    'Total number of errors',
    ['error_type']
)

# Отслеживание активных звонков
active_calls_tracker = {}  # {(user1, user2): start_time}


async def metrics_handler(request):
    """Prometheus metrics endpoint"""
    try:
        metrics_output = generate_latest(REGISTRY)
        return web.Response(
            body=metrics_output,
            headers={'Content-Type': CONTENT_TYPE_LATEST}
        )
    except Exception as e:
        logger.error(f"Error generating metrics: {e}")
        return web.Response(
            text=f"Error generating metrics: {str(e)}",
            status=500
        )


async def health_handler(request):
    """Health check endpoint"""
    return web.json_response({
        'status': 'healthy',
        'active_users': len(users),
        'active_calls': len(active_calls_tracker)
    })


async def websocket_handler(request):
    """Обработчик WebSocket соединений"""
    ws = web.WebSocketResponse()
    await ws.prepare(request)
    
    WEBSOCKET_CONNECTIONS.inc()
    username = None
    
    try:
        async for msg in ws:
            if msg.type == web.WSMsgType.TEXT:
                try:
                    data = json.loads(msg.data)
                    message_type = data.get('type')
                    MESSAGES_TOTAL.labels(message_type=message_type).inc()
                    
                    # Регистрация пользователя
                    if message_type == 'login':
                        username = data.get('username')
                        if username in users:
                            await ws.send_json({
                                'type': 'error',
                                'message': 'Username already taken'
                            })
                            ERRORS_TOTAL.labels(error_type='duplicate_username').inc()
                            await ws.close()
                            return
                        
                        users[username] = ws
                        USERS_CONNECTED.set(len(users))
                        await ws.send_json({
                            'type': 'login_success',
                            'username': username
                        })
                        logger.info(f"User {username} connected. Total users: {len(users)}")
                    
                    # Инициация звонка
                    elif message_type == 'call':
                        target = data.get('target')
                        call_type = data.get('callType')
                        
                        if target not in users:
                            await ws.send_json({
                                'type': 'error',
                                'message': f'User {target} not found'
                            })
                            ERRORS_TOTAL.labels(error_type='user_not_found').inc()
                        else:
                            target_ws = users[target]
                            await target_ws.send_json({
                                'type': 'incoming_call',
                                'from': username,
                                'callType': call_type
                            })
                            CALLS_TOTAL.labels(call_type=call_type).inc()
                            
                            # Начинаем отслеживать звонок
                            call_key = tuple(sorted([username, target]))
                            active_calls_tracker[call_key] = datetime.now()
                            ACTIVE_CALLS.set(len(active_calls_tracker))
                            
                            logger.info(f"Call from {username} to {target} ({call_type})")
                    
                    # WebRTC сигнализация - Offer
                    elif message_type == 'offer':
                        target = data.get('target')
                        offer = data.get('offer')
                        
                        if target in users:
                            target_ws = users[target]
                            await target_ws.send_json({
                                'type': 'offer',
                                'from': username,
                                'offer': offer
                            })
                            logger.info(f"Offer from {username} to {target}")
                    
                    # WebRTC сигнализация - Answer
                    elif message_type == 'answer':
                        target = data.get('target')
                        answer = data.get('answer')
                        
                        if target in users:
                            target_ws = users[target]
                            await target_ws.send_json({
                                'type': 'answer',
                                'from': username,
                                'answer': answer
                            })
                            logger.info(f"Answer from {username} to {target}")
                    
                    # WebRTC сигнализация - ICE Candidate
                    elif message_type == 'ice-candidate':
                        target = data.get('target')
                        candidate = data.get('candidate')
                        
                        if target in users:
                            target_ws = users[target]
                            await target_ws.send_json({
                                'type': 'ice-candidate',
                                'from': username,
                                'candidate': candidate
                            })
                    
                    # Отклонение звонка
                    elif message_type == 'decline':
                        target = data.get('target')
                        
                        if target in users:
                            target_ws = users[target]
                            await target_ws.send_json({
                                'type': 'call_declined',
                                'from': username
                            })
                            
                            # Завершаем отслеживание звонка
                            call_key = tuple(sorted([username, target]))
                            if call_key in active_calls_tracker:
                                start_time = active_calls_tracker.pop(call_key)
                                duration = (datetime.now() - start_time).total_seconds()
                                CALL_DURATION.observe(duration)
                                ACTIVE_CALLS.set(len(active_calls_tracker))
                            
                            logger.info(f"Call declined by {username}")
                    
                    # Завершение звонка
                    elif message_type == 'end_call':
                        target = data.get('target')
                        
                        if target in users:
                            target_ws = users[target]
                            await target_ws.send_json({
                                'type': 'call_ended',
                                'from': username
                            })
                            
                            # Завершаем отслеживание звонка
                            call_key = tuple(sorted([username, target]))
                            if call_key in active_calls_tracker:
                                start_time = active_calls_tracker.pop(call_key)
                                duration = (datetime.now() - start_time).total_seconds()
                                CALL_DURATION.observe(duration)
                                ACTIVE_CALLS.set(len(active_calls_tracker))
                            
                            logger.info(f"Call ended by {username}")
                
                except json.JSONDecodeError:
                    logger.error("Invalid JSON received")
                    ERRORS_TOTAL.labels(error_type='invalid_json').inc()
                except Exception as e:
                    logger.error(f"Error processing message: {e}")
                    ERRORS_TOTAL.labels(error_type='processing_error').inc()
            
            elif msg.type == web.WSMsgType.ERROR:
                logger.error(f'WebSocket error: {ws.exception()}')
                ERRORS_TOTAL.labels(error_type='websocket_error').inc()
    
    finally:
        # Удаление пользователя при отключении
        if username and username in users:
            del users[username]
            USERS_CONNECTED.set(len(users))
            
            # Удаляем все звонки пользователя
            to_remove = [k for k in active_calls_tracker.keys() if username in k]
            for call_key in to_remove:
                start_time = active_calls_tracker.pop(call_key)
                duration = (datetime.now() - start_time).total_seconds()
                CALL_DURATION.observe(duration)
            ACTIVE_CALLS.set(len(active_calls_tracker))
            
            logger.info(f"User {username} disconnected. Total users: {len(users)}")
        
        WEBSOCKET_CONNECTIONS.dec()
    
    return ws


async def index_handler(request):
    """Возвращает HTML страницу"""
    with open('index.html', 'r', encoding='utf-8') as f:
        return web.Response(text=f.read(), content_type='text/html')


def main():
    """Запуск сервера"""
    app = web.Application()
    
    # Роуты
    app.router.add_get('/', index_handler)
    app.router.add_get('/ws', websocket_handler)
    app.router.add_get('/metrics', metrics_handler)
    app.router.add_get('/health', health_handler)
    
    # CORS для разработки
    async def cors_middleware(app, handler):
        async def middleware_handler(request):
            response = await handler(request)
            response.headers['Access-Control-Allow-Origin'] = '*'
            response.headers['Access-Control-Allow-Methods'] = 'GET, POST, OPTIONS'
            response.headers['Access-Control-Allow-Headers'] = '*'
            return response
        return middleware_handler
    
    app.middlewares.append(cors_middleware)
    
    # Запуск
    logger.info("Starting WebRTC Signaling Server on http://localhost:8000")
    logger.info("Metrics available at http://localhost:8000/metrics")
    web.run_app(app, host='0.0.0.0', port=8000)


if __name__ == '__main__':
    main()