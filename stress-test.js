import http from 'k6/http';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';

// 定义自定义指标
const appointmentSuccessCounter = new Counter('custom_appointment_success_count');
const appointmentLatencyTrend = new Trend('custom_appointment_latency_md');

export const options = {
    scenarios: {
        default: {
            executor: 'ramping-vus',
            stages: [
                { duration: '10s', target: 300 },  // 10秒内快速冲到 300 并发
                { duration: '30s', target: 800 },  // 30秒内饱和轰炸，冲到 800 并发
                { duration: '10s', target: 0 },    // 10秒内收尾
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
    const url = 'http://host.docker.internal:8888/v1/appointment'; // Windows Docker 连宿主机推荐用这个
    const payload = JSON.stringify({
        userId: Math.floor(Math.random() * 1000000) + 10000, // 随机生成用户ID，防止撞重
        scheduleId: 99,
        orderNo: `STRESS_${Date.now()}_${Math.floor(Math.random() * 1000)}`,
    });

    const params = {
        headers: { 'Content-Type': 'application/json' },
        tags: { name: 'PostAppointment' },
    };

    const res = http.post(url, payload, params);

    // 统计指标
    check(res, {
        'is status 200': (r) => r.status === 200,
    });

    appointmentSuccessCounter.add(1);
    appointmentLatencyTrend.add(res.timings.duration);

    // 【核心修改】：彻底注释掉 sleep，让每个 VU 陷入疯狂的无休止死循环请求中！
    // sleep(0.1);
}