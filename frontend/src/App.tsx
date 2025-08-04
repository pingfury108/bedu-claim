import ClueClaimingComponent from './ClueClaimingComponent';

function App() {
    return (
        <div className="min-h-screen bg-base-200 p-4">
            <div className="max-w-4xl mx-auto">
                <div className="text-center mb-6">
                    <h1 className="text-2xl font-bold text-primary">任务自动认领系统</h1>
                </div>
                <ClueClaimingComponent />
            </div>
        </div>
    )
}

export default App
