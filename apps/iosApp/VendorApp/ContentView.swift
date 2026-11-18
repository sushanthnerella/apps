import SwiftUI
import VendorApp
struct VendorContentView: UIViewControllerRepresentable {
    func makeUIViewController(context: Context) -> UIViewController { MainViewControllerKt.MainViewController() }
    func updateUIViewController(_ uiViewController: UIViewController, context: Context) {}
}
