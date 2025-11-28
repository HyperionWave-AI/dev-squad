const WebSocket = require('ws');

const SESSION_ID = '6929822812d1d05be760418d';
const USER_ID = 'dev-user';
const COMPANY_ID = 'dev-company';

const url = `ws://localhost:5555/api/v1/chat/stream?sessionId=${SESSION_ID}&userId=${USER_ID}&companyId=${COMPANY_ID}`;

const topics = [
    'error handling',
    'concurrency patterns',
    'database transactions',
    'caching strategies',
    'monitoring and observability'
];

let messageCount = 0;
const MAX_MESSAGES = 5;

function sendMessage(ws) {
    messageCount++;
    const topic = topics[messageCount - 1];
    console.log(`\n\n========== Sending message ${messageCount}/${MAX_MESSAGES} ==========`);
    ws.send(JSON.stringify({
        type: 'message',
        content: `Message ${messageCount}: Explain in great detail with examples how to implement ${topic} in Go distributed systems. Include code snippets and best practices.`
    }));
}

console.log('Connecting to:', url);
const ws = new WebSocket(url);

ws.on('open', () => {
    console.log('Connected!');
    sendMessage(ws);
});

ws.on('message', (data) => {
    try {
        const msg = JSON.parse(data.toString());
        if (msg.type === 'system_notification') {
            console.log('\n\n🔔🔔🔔 NOTIFICATION 🔔🔔🔔');
            console.log(JSON.stringify(msg.notification, null, 2));
            console.log('🔔🔔🔔🔔🔔🔔🔔🔔🔔🔔🔔🔔\n');
        } else if (msg.type === 'token') {
            process.stdout.write(msg.content || '');
        } else if (msg.type === 'done') {
            console.log('\n✅ Response complete');
            if (messageCount < MAX_MESSAGES) {
                setTimeout(() => sendMessage(ws), 1000);
            } else {
                console.log('\n\n🏁 All messages sent! Check server logs for compaction.');
                ws.close();
            }
        } else if (msg.type === 'error') {
            console.log('\n❌ Error:', msg.error);
        }
    } catch(e) {}
});

ws.on('error', (err) => console.error('Error:', err.message));
ws.on('close', () => { console.log('\nClosed'); process.exit(0); });
setTimeout(() => { ws.close(); process.exit(0); }, 300000);
