import axios, {AxiosInstance, AxiosRequestConfig} from "axios";

export class ApiClient {
    private client: AxiosInstance;

    constructor(baseURL: string) {
        this.client = axios.create({
            baseURL,
            headers: {
                "Content-Type": "application/json",
            },
        });
    }

    async get<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
        const res = await this.client.get<T>(url, config);
        return res.data;
    }

    async post<T, B = unknown>(
        url: string,
        body?: B,
        config?: AxiosRequestConfig
    ): Promise<T> {
        const res = await this.client.post<T>(url, body, config);
        return res.data;
    }

    async put<T, B = unknown>(
        url: string,
        body?: B,
        config?: AxiosRequestConfig
    ): Promise<T> {
        const res = await this.client.put<T>(url, body, config);
        return res.data;
    }

    async patch<T, B = unknown>(
        url: string,
        body?: B,
        config?: AxiosRequestConfig
    ): Promise<T> {
        const res = await this.client.patch<T>(url, body, config);
        return res.data;
    }

    async delete<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
        const res = await this.client.delete<T>(url, config);
        return res.data;
    }
}

export const api = new ApiClient("");