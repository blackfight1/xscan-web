import axios from 'axios'

const api = axios.create({
    baseURL: '/api',
    timeout: 30000,
    headers: {
        'Content-Type': 'application/json',
    },
})

// Set auth token from localStorage
const token = localStorage.getItem('xscan_token')
if (token) {
    api.defaults.headers.common['Authorization'] = `Bearer ${token}`
}

// Set token
export function setToken(token) {
    localStorage.setItem('xscan_token', token)
    api.defaults.headers.common['Authorization'] = `Bearer ${token}`
}

// Get token
export function getToken() {
    return localStorage.getItem('xscan_token') || ''
}

// Response interceptor
api.interceptors.response.use(
    (response) => response.data,
    (error) => {
        if (error.response?.status === 401) {
            localStorage.removeItem('xscan_token')
            window.location.reload()
        }
        return Promise.reject(error)
    }
)

// Tasks API
export function createTask(rootDomain) {
    return api.post('/tasks', { root_domain: rootDomain })
}

export function getTasks() {
    return api.get('/tasks')
}

export function getTask(id) {
    return api.get(`/tasks/${id}`)
}

export function deleteTask(id) {
    return api.delete(`/tasks/${id}`)
}

export function getReport(id) {
    return api.get(`/tasks/${id}/report`)
}

export default api
