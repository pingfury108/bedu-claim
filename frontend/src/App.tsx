import ClueClaimingComponent from './ClueClaimingComponent';

function App() {
    return (
        <div className="min-h-screen bg-base-200 p-4">
            <div className="max-w-4xl mx-auto">
                <div className="text-center mb-6">
                    <h1 className="text-2xl font-bold text-primary">自动认领系统</h1>
                    <p className="text-gray-600">配置并启动自动认领任务</p>
                </div>
                <ClueClaimingComponent />
            </div>
        </div>
    )
}

export default App
