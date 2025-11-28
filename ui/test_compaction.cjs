const WebSocket = require('ws');

const SESSION_ID = '6929822812d1d05be760418d';
const USER_ID = 'dev-user';
const COMPANY_ID = 'dev-company';

const url = `ws://localhost:5555/api/v1/chat/stream?sessionId=${SESSION_ID}&userId=${USER_ID}&companyId=${COMPANY_ID}`;

console.log('Connecting to:', url);

const ws = new WebSocket(url);

ws.on('open', () => {
    console.log('Connected! Sending message...');
    ws.send(JSON.stringify({
        type: 'message',
        content: 'Tell me a detailed story about building distributed systems with Go. Include architecture patterns, error handling, concurrency models, database design, caching strategies, monitoring, and deployment. Make it comprehensive.'
    }));
});

ws.on('message', (data) => {
    try {
        const msg = JSON.parse(data.toString());
        if (msg.type === 'system_notification') {
            console.log('\n\n🔔 NOTIFICATION:', JSON.stringify(msg.notification, null, 2));
        } else if (msg.type === 'token') {
            process.stdout.write(msg.content || '');
        } else if (msg.type === 'done') {
            console.log('\n\n✅ Response complete');
            ws.close();
        } else if (msg.type === 'error') {
            console.log('\n❌ Error:', msg.error);
        } else {
            console.log('\n[' + msg.type + ']');
        }
    } catch(e) {
        console.log('Raw:', data.toString().substring(0, 100));
    }
});

ws.on('error', (err) => {
    console.error('WebSocket Error:', err.message);
});

ws.on('close', () => {
    console.log('\nConnection closed');
    process.exit(0);
});

setTimeout(() => {
    console.log('\nTimeout - closing');
    ws.close();
    process.exit(0);
}, 120000);
