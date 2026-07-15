import http from 'k6/http';
import {check} from 'k6';
import {Counter, Trend} from 'k6/metrics';

// 定义自定义指标
const appointmentSuccessCounter = new Counter('custom_appointment_success_count');
const appointmentLatencyTrend = new Trend('custom_appointment_latency_md');

export const options = {
    scenarios: {
        default: {
            executor: 'ramping-vus',
            stages: [
                {duration: '10s', target: 300},  // 10秒内快速冲到 300 并发
                {duration: '30s', target: 800},  // 30秒内饱和轰炸，冲到 800 并发
                {duration: '10s', target: 0},    // 10秒内收尾
            ],
            gracefulRampDown: '10s',
            gracefulStop: '10s',
        },
    },
    thresholds: {
        // 我们可以暂时把门禁放宽，专门看它能高到什么程度
        'http_req_duration{name:PostAppointment}': ['p(99)<500'],
        'http_req_failed': ['rate<0.05'],
    },
};

export default function () {
    const url = 'http://host.docker.internal:8888/v1/appointment';

    // 基于当前的 VU ID 和迭代次数，生成一个唯一的、可预测的 UserId
    const userId = (__VU * 1000) + __ITER;

    const payload = JSON.stringify({
        userId: userId,
        scheduleId: 99,
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
            'X-User-Id': userId.toString() // 💡 注入这个 Header
        },
        tags: {name: 'PostAppointment'},
    };

    const res = http.post(url, payload, params);

    check(res, {
        'is status 200': (r) => r.status === 200,
        // 💡 验证灰度返回：如果是尾数为 0 的用户，Msg 应该包含 [Canary]
        'canary route correct': (r) => {
            if (userId % 10 === 0) {
                return r.json().msg.includes('[Canary]');
            }
            return !r.json().msg.includes('[Canary]');
        }
    });
}